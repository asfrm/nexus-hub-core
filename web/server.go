package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"nexus-hub-core/moonraker"
	"nexus-hub-core/slicer"
	"os"
	"path/filepath"
	"strconv"
)

func StartWebEngine(database *db.Database, tx chan api.NexusPacket, port int) {
	mux := http.NewServeMux()

	// --- 1. СЕРВИСНЫЕ РОУТЫ ---
	mux.HandleFunc("/server/files/upload", slicer.UploadGCodeHandler(database, tx))
	mux.HandleFunc("/ws", HandleWS(tx))

	// --- 2. VAULT & JOB LIST (Очереди) ---

	// Хранилище (PENDING)
	mux.HandleFunc("/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		jobs, _ := database.GetPendingJobs()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
	})

	// Активный список (ASSIGNED / PRINTING)
	mux.HandleFunc("/api/jobs/list", func(w http.ResponseWriter, r *http.Request) {
		jobs, _ := database.GetJobList()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
	})

	// Назначение задачи (Assign) + ТРИГГЕР ПЕЧАТИ
	mux.HandleFunc("/api/jobs/assign", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			JobID     int64 `json:"job_id"`
			ClusterID int64 `json:"cluster_id"`
			PrinterID int64 `json:"printer_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}

		database.AssignJob(req.JobID, req.ClusterID, req.PrinterID)

		if req.PrinterID > 0 {
			job, _ := database.GetJobByID(req.JobID)
			clusters, _ := database.GetClustersWithPrinters()

			var targetIP string
			for _, c := range clusters {
				for _, p := range c.Printers {
					if p.ID == req.PrinterID {
						targetIP = p.IPAddress
					}
				}
			}

			if targetIP != "" {
				filePath := filepath.Join("gcode_vault", job.Filename)
				go func(ip, path string, id int64) {
					err := moonraker.UploadAndPrint(ip, path)
					if err == nil {
						database.UpdateJobStatus(id, "PRINTING")
						database.UpdatePrinterStatus(req.PrinterID, "PRINTING")
					}
				}(targetIP, filePath, req.JobID)
			}
		}

		w.Write([]byte(`{"success":true}`))
	})

	// --- 3. CLUSTERS & PRINTERS ---

	// ЕДИНСТВЕННЫЙ РОУТ ДЛЯ СПИСКА КЛАСТЕРОВ
	mux.HandleFunc("/api/clusters", func(w http.ResponseWriter, r *http.Request) {
		clusters, err := database.GetClustersWithPrinters()
		if err != nil {
			fmt.Printf("❌ Ошибка при получении кластеров: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("📦 Отправка данных на фронт: %d кластеров\n", len(clusters))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clusters)
	})

	mux.HandleFunc("/api/clusters/create", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name  string `json:"name"`
			Model string `json:"target_model"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		database.CreateCluster(req.Name, req.Model)
		w.Write([]byte(`{"success":true}`))
	})

	mux.HandleFunc("/api/clusters/delete", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		database.DeleteCluster(id)
		w.Write([]byte(`{"success":true}`))
	})

	mux.HandleFunc("/api/clusters/toggle", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		enabled, _ := strconv.ParseBool(r.URL.Query().Get("enabled"))
		database.ToggleCluster(id, enabled)
		w.Write([]byte(`{"success":true}`))
	})

	mux.HandleFunc("/api/printers/add", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ClusterID int64  `json:"cluster_id"`
			Name      string `json:"name"`
			IP        string `json:"ip_address"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		database.AddPrinter(req.ClusterID, req.Name, req.IP)
		w.Write([]byte(`{"success":true}`))
	})

	mux.HandleFunc("/api/printers/control", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		action := r.URL.Query().Get("action")

		clusters, _ := database.GetClustersWithPrinters()
		for _, c := range clusters {
			for _, p := range c.Printers {
				if p.ID == id {
					go moonraker.SendCommand(p.IPAddress, action)
				}
			}
		}
		w.Write([]byte(`{"success":true}`))
	})

	// --- 4. SETUP WIZARD ---
	mux.HandleFunc("/api/setup/analyze-gcode", func(w http.ResponseWriter, r *http.Request) {
		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "No file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		tempPath := filepath.Join("gcode_vault", "temp_"+handler.Filename)
		dst, _ := os.Create(tempPath)
		io.Copy(dst, file)
		dst.Close()
		meta := slicer.ParseGCodeMetadata(tempPath)
		os.Remove(tempPath)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"detected_model": meta.PrinterModel, "filament_type": meta.FilamentType})
	})

	mux.HandleFunc("/api/setup/unassigned", func(w http.ResponseWriter, r *http.Request) {
		models, _ := database.GetUnassignedModels()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models)
	})

	// --- 5. ЗАПУСК СЕРВЕРА ---
	mux.Handle("/", http.FileServer(http.Dir("ui")))
	handlerWithCORS := middlewareCORS(mux)

	fmt.Printf("📡 Nexus Hub API запущен на порту: %d\n", port)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), handlerWithCORS); err != nil {
		fmt.Printf("Критическая ошибка сервера: %v\n", err)
	}
}

func middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
