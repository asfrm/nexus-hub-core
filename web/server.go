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
	"time"
)

func StartWebEngine(database *db.Database, hub *api.Hub, port int) {
	mux := http.NewServeMux()

	// --- 1. SERVICE ROUTES ---
	mux.HandleFunc("/server/files/upload", slicer.UploadGCodeHandler(database, hub))
	mux.HandleFunc("/ws", HandleWS(hub))

	// --- 1.5 NEW: Printer Status Endpoint ---
	mux.HandleFunc("/api/printers/status", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid printer ID", http.StatusBadRequest)
			return
		}

		clusters, err := database.GetClustersWithPrinters()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var printerIP string
		for _, c := range clusters {
			for _, p := range c.Printers {
				if p.ID == id {
					printerIP = p.IPAddress
					break
				}
			}
		}

		if printerIP == "" {
			http.Error(w, "Printer not found", http.StatusNotFound)
			return
		}

		status, err := moonraker.GetStatus(printerIP)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	// --- 1.6 NEW: Completed Jobs ---
	mux.HandleFunc("/api/jobs/completed", func(w http.ResponseWriter, r *http.Request) {
		rows, err := database.Conn.Query(`
			SELECT id, filename, filament_type, printer_model, estimated_time_s, status, 
			       target_cluster_id, target_printer_id, created_at 
			FROM print_jobs 
			WHERE status = 'COMPLETED' 
			ORDER BY created_at DESC 
			LIMIT 50
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		jobs := []db.PrintJob{}
		for rows.Next() {
			var j db.PrintJob
			if err := rows.Scan(&j.ID, &j.Filename, &j.Filament, &j.PrinterModel,
				&j.EstimatedTimeS, &j.Status, &j.TargetClusterID, &j.TargetPrinterID, &j.CreatedAt); err == nil {
				jobs = append(jobs, j)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
	})

	// --- 1.7 NEW: System Stats ---
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		pending, _ := database.GetPendingJobs()
		active, _ := database.GetJobList()
		clusters, _ := database.GetClustersWithPrinters()

		var totalPrinters, activePrinters int
		for _, c := range clusters {
			for _, p := range c.Printers {
				totalPrinters++
				if p.Status == "PRINTING" {
					activePrinters++
				}
			}
		}

		stats := map[string]interface{}{
			"pending_jobs":    len(pending),
			"active_jobs":     len(active),
			"total_clusters":  len(clusters),
			"total_printers":  totalPrinters,
			"active_printers": activePrinters,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	// --- 2. VAULT & JOB LIST (Queues) ---

	// Storage (PENDING)
	mux.HandleFunc("/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		jobs, err := database.GetPendingJobs()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
	})

	// Active list (ASSIGNED / PRINTING)
	mux.HandleFunc("/api/jobs/list", func(w http.ResponseWriter, r *http.Request) {
		jobs, err := database.GetJobList()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
	})

	// Job assignment + PRINT TRIGGER
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

		if err := database.AssignJob(req.JobID, req.ClusterID, req.PrinterID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if req.PrinterID > 0 {
			job, err := database.GetJobByID(req.JobID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			clusters, err := database.GetClustersWithPrinters()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

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

						// Log to system
						hub.Send(api.NexusPacket{
							EventType: "SYSTEM_LOG",
							Log: &api.LogPayload{
								Module:    "WEB",
								Level:     "INFO",
								Message:   fmt.Sprintf("Job '%s' started on printer ID %d", job.Filename, req.PrinterID),
								Timestamp: time.Now(),
							},
						})
					} else {
						hub.Send(api.NexusPacket{
							EventType: "SYSTEM_LOG",
							Log: &api.LogPayload{
								Module:    "WEB",
								Level:     "ERROR",
								Message:   fmt.Sprintf("Failed to start job '%s': %v", job.Filename, err),
								Timestamp: time.Now(),
							},
						})
					}
				}(targetIP, filePath, req.JobID)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})

	// --- 3. CLUSTERS & PRINTERS ---

	// Get all clusters
	mux.HandleFunc("/api/clusters", func(w http.ResponseWriter, r *http.Request) {
		clusters, err := database.GetClustersWithPrinters()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clusters)
	})

	mux.HandleFunc("/api/clusters/create", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name  string `json:"name"`
			Model string `json:"target_model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}
		if err := database.CreateCluster(req.Name, req.Model); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})

	mux.HandleFunc("/api/clusters/delete", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}
		if err := database.DeleteCluster(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})

	mux.HandleFunc("/api/clusters/toggle", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}
		enabled, err := strconv.ParseBool(r.URL.Query().Get("enabled"))
		if err != nil {
			http.Error(w, "Invalid enabled value", http.StatusBadRequest)
			return
		}
		if err := database.ToggleCluster(id, enabled); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})

	mux.HandleFunc("/api/printers/add", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ClusterID int64  `json:"cluster_id"`
			Name      string `json:"name"`
			IP        string `json:"ip_address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}
		if err := database.AddPrinter(req.ClusterID, req.Name, req.IP); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})

	mux.HandleFunc("/api/printers/control", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}
		action := r.URL.Query().Get("action")

		clusters, err := database.GetClustersWithPrinters()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, c := range clusters {
			for _, p := range c.Printers {
				if p.ID == id {
					go func(printerIP string) {
						if err := moonraker.SendCommand(printerIP, action); err != nil {
							hub.Send(api.NexusPacket{
								EventType: "SYSTEM_LOG",
								Log: &api.LogPayload{
									Module:    "WEB",
									Level:     "ERROR",
									Message:   fmt.Sprintf("Failed to send command '%s' to printer: %v", action, err),
									Timestamp: time.Now(),
								},
							})
						}
					}(p.IPAddress)
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})

	// --- 4. SETUP WIZARD ---
	mux.HandleFunc("/api/setup/analyze-gcode", func(w http.ResponseWriter, r *http.Request) {
		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "No file provided", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Validate file extension
		ext := filepath.Ext(handler.Filename)
		if ext != ".gcode" && ext != ".g" {
			http.Error(w, "Invalid file type. Only .gcode files are allowed", http.StatusBadRequest)
			return
		}

		tempPath := filepath.Join("gcode_vault", "temp_"+handler.Filename)
		dst, err := os.Create(tempPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if _, err := io.Copy(dst, file); err != nil {
			dst.Close()
			os.Remove(tempPath)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		dst.Close()

		meta := slicer.ParseGCodeMetadata(tempPath)
		os.Remove(tempPath)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"detected_model": meta.PrinterModel,
			"filament_type":  meta.FilamentType,
		})
	})

	mux.HandleFunc("/api/setup/unassigned", func(w http.ResponseWriter, r *http.Request) {
		models, err := database.GetUnassignedModels()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models)
	})

	// --- 5. START SERVER ---
	mux.Handle("/", http.FileServer(http.Dir("ui")))
	handlerWithCORS := middlewareCORS(mux)

	fmt.Printf("📡 Nexus Hub API started on port: %d\n", port)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), handlerWithCORS); err != nil {
		fmt.Printf("Critical server error: %v\n", err)
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
