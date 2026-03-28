package web

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"nexus-hub-core/moonraker"
	"nexus-hub-core/parser"
	"nexus-hub-core/printer"
	"nexus-hub-core/printer/bambu"
	"nexus-hub-core/scanner"
	"nexus-hub-core/slicer"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func StartWebEngine(database *db.Database, hub *api.Hub, port int) {
	mux := http.NewServeMux()

	// --- 1. SERVICE ROUTES ---
	mux.HandleFunc("/server/files/upload", slicer.UploadGCodeHandler(database, hub))
	mux.HandleFunc("/ws", HandleWS(hub))

	// --- 1.5 SLICER GRABBER API (OctoPrint-compatible) ---
	mux.HandleFunc("/api/files/local", handleSlicerUpload(database, hub))

	// --- 1.6 HEALTH CHECK ---
	startTime := time.Now()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		health := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "2.0.3",
			"uptime":    time.Since(startTime).String(),
			"database":  "connected",
			"printers":  0,
		}

		// Check database
		if err := database.Conn.Ping(); err != nil {
			health["database"] = "disconnected"
			health["status"] = "degraded"
		}

		// Count printers
		clusters, _ := database.GetClustersWithPrinters()
		for _, c := range clusters {
			health["printers"] = health["printers"].(int) + len(c.Printers)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	})

	// --- 1.5 NEW: Completed Jobs ---
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

	// --- 1.8 NEW: Analytics Dashboard ---
	mux.HandleFunc("/api/analytics", func(w http.ResponseWriter, r *http.Request) {
		analytics, err := database.GetSystemAnalytics()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Add wear stats
		wearStats, err := database.GetPrinterWearStats()
		if err == nil {
			analytics["printer_wear"] = wearStats
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(analytics)
	})

	// --- 1.9 NEW: Print History ---
	mux.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil {
				limit = parsed
			}
		}

		history, err := database.GetPrintHistory(limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(history)
	})

	// --- 1.10 NEW: Wear Stats ---
	mux.HandleFunc("/api/printers/wear", func(w http.ResponseWriter, r *http.Request) {
		wearStats, err := database.GetPrinterWearStats()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wearStats)
	})

	// --- 1.11 NEW: Filament Spools ---
	mux.HandleFunc("/api/spools", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			spools, err := database.GetSpools()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(spools)

		case "POST":
			var req struct {
				Material    string  `json:"material"`
				Color       string  `json:"color"`
				ColorHex    string  `json:"color_hex"`
				TotalWeight float32 `json:"total_weight_g"`
				AssignedTo  int64   `json:"assigned_printer_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Bad JSON", http.StatusBadRequest)
				return
			}
			if err := database.AddSpool(req.Material, req.Color, req.ColorHex, req.TotalWeight, req.TotalWeight, req.AssignedTo); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
		}
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
					// Simple Moonraker upload using HTTP
					err := uploadToMoonraker(ip, path)
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
			ClusterID   int64  `json:"cluster_id"`
			Name        string `json:"name"`
			IP          string `json:"ip_address"`
			PrinterType string `json:"printer_type"`
			Serial      string `json:"serial"`      // Bambu serial (manual fallback)
			AccessCode  string `json:"access_code"` // Bambu 8-char code (manual fallback)
			Username    string `json:"username"`
			Password    string `json:"password"`
			AccessToken string `json:"access_token"`
			Extra       string `json:"extra"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}
		if err := database.AddPrinter(req.ClusterID, req.Name, req.IP, req.PrinterType, req.Serial, req.AccessCode, req.Username, req.Password, req.AccessToken, "", req.Extra); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("[API] Added printer: %s (%s) at %s", req.Name, req.Serial, req.IP)
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
					// HARDWARE SAFETY LOCK: Check if printer is printing before sending G-code
					if action == "g28" && (p.Status == "PRINTING" || p.Status == "PAUSED") {
						hub.Send(api.NexusPacket{
							EventType: "SYSTEM_LOG",
							Log: &api.LogPayload{
								Module:    "WEB",
								Level:     "ERROR",
								Message:   fmt.Sprintf("Hardware safety lock: Cannot home while printing (status: %s)", p.Status),
								Timestamp: time.Now(),
							},
						})
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusForbidden)
						json.NewEncoder(w).Encode(map[string]string{
							"error":  "Cannot send G-code while printer is printing or paused",
							"status": p.Status,
						})
						return
					}

					if p.PrinterType == "BAMBULAB" {
						// Use MQTT for Bambu printers
						go sendBambuCommand(p, action, hub)
					} else {
						// Use HTTP for Moonraker
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

	// --- 6. BAMBU LAB CLOUD API ---

	// Bambu Login URL (for OAuth-style flow)
	mux.HandleFunc("/api/bambu/login-url", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API DEBUG] Bambu login URL requested")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"login_url":    bambu.GetLoginURL(),
			"instructions": "1. Click the URL to open Bambu Lab login page\n2. After logging in, go to your profile\n3. Copy your 'User Access Token'\n4. Use the /api/bambu/token endpoint with the token",
		})
	})

	// Bambu Token-based Login (OAuth-style)
	mux.HandleFunc("/api/bambu/token", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API DEBUG] Bambu token login request received")

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Email       string `json:"email"`
			Username    string `json:"username"`
			UID         string `json:"uid"`
			AccessToken string `json:"access_token"`
			Region      string `json:"region"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[API DEBUG] Bambu token: Bad JSON - %v", err)
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}

		log.Printf("[API DEBUG] Bambu token: Saving token account for user: %s (uid: %s)", req.Email, req.UID)

		// Save account with token
		err := database.SaveBambuTokenAccount(req.Email, req.Username, req.UID, req.AccessToken, req.Region)
		if err != nil {
			log.Printf("[API DEBUG] Bambu token: Failed to save account - %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("[API DEBUG] Bambu token: Account saved successfully")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Token saved successfully",
		})
	})

	// Bambu Callback (for OAuth redirect)
	mux.HandleFunc("/api/bambu/callback", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API DEBUG] Bambu callback received")

		token := r.URL.Query().Get("token")
		uid := r.URL.Query().Get("uid")
		username := r.URL.Query().Get("username")
		email := r.URL.Query().Get("email")
		region := r.URL.Query().Get("region")

		if token == "" {
			http.Error(w, "Token required", http.StatusBadRequest)
			return
		}

		log.Printf("[API DEBUG] Bambu callback: Saving account from OAuth callback")

		err := database.SaveBambuTokenAccount(email, username, uid, token, region)
		if err != nil {
			log.Printf("[API DEBUG] Bambu callback: Failed to save account - %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Redirect to success page
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>Authentication Successful</title>
				<script>
					window.opener.postMessage({ success: true, email: '` + email + `' }, '*');
					window.close();
				</script>
			</head>
			<body>
				<h1>Authentication Successful!</h1>
				<p>You can close this window and return to the application.</p>
			</body>
			</html>
		`))
	})

	// Bambu Legacy Password Login (still supported)
	mux.HandleFunc("/api/bambu/login", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API DEBUG] Bambu legacy password login request received")
		if r.Method != "POST" {
			log.Printf("[API DEBUG] Bambu login: Method not allowed")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Region   string `json:"region"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[API DEBUG] Bambu login: Bad JSON - %v", err)
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}

		log.Printf("[API DEBUG] Bambu login: Saving account for user: %s (region: %s)", req.Email, req.Region)

		// Save account (password will be hashed in production)
		err := database.SaveBambuAccount(req.Email, req.Password, "", "", req.Region, time.Now())
		if err != nil {
			log.Printf("[API DEBUG] Bambu login: Failed to save account - %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("[API DEBUG] Bambu login: Account saved successfully")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Account saved. Authentication will occur on first connection.",
		})
	})

	// Bambu Verify 2FA
	mux.HandleFunc("/api/bambu/verify-2fa", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API DEBUG] Bambu 2FA verification request received")
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Email string `json:"email"`
			Code  string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[API DEBUG] Bambu 2FA: Bad JSON - %v", err)
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}

		log.Printf("[API DEBUG] Bambu 2FA: Verifying code for user: %s", req.Email)

		// For now, just mark TFA as verified
		err := database.SetBambuTFA(req.Email, true, "")
		if err != nil {
			log.Printf("[API DEBUG] Bambu 2FA: Failed to verify - %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("[API DEBUG] Bambu 2FA: Verification successful")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	})

	// Bambu Discover Printers
	mux.HandleFunc("/api/bambu/discover", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API DEBUG] Bambu discover printers request received")

		email := r.URL.Query().Get("email")
		if email == "" {
			log.Printf("[API DEBUG] Bambu discover: No email provided, using default account")
			// Get default account
			acc, err := database.GetDefaultBambuAccount()
			if err != nil {
				log.Printf("[API DEBUG] Bambu discover: Failed to get default account - %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if acc == nil {
				log.Printf("[API DEBUG] Bambu discover: No default account configured")
				http.Error(w, "No Bambu account configured", http.StatusNotFound)
				return
			}
			email = acc.Email
			log.Printf("[API DEBUG] Bambu discover: Using default account: %s", email)
		}

		acc, err := database.GetBambuAccount(email)
		if err != nil {
			log.Printf("[API DEBUG] Bambu discover: Failed to get account - %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if acc == nil {
			log.Printf("[API DEBUG] Bambu discover: Account not found: %s", email)
			http.Error(w, "Account not found", http.StatusNotFound)
			return
		}

		log.Printf("[API DEBUG] Bambu discover: Creating provider for user: %s (region: %s)", email, acc.Region)

		// Create a temporary provider to discover printers
		config := printer.PrinterConfig{
			Username:    email,
			Password:    acc.PasswordHash, // In production, decrypt this
			AccessToken: acc.AccessToken,
			Extra: map[string]string{
				"region": acc.Region,
			},
		}

		provider := printer.NewBambuProvider(config)
		ctx := context.Background()

		// Authenticate first
		log.Printf("[API DEBUG] Bambu discover: Attempting authentication...")
		if err := provider.Authenticate(ctx); err != nil {
			log.Printf("[API DEBUG] Bambu discover: Authentication failed - %v", err)
			// Check if 2FA is required
			if provider.IsTFAPending() {
				log.Printf("[API DEBUG] Bambu discover: 2FA verification required")
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success":      false,
					"tfa_required": true,
					"message":      "2FA verification required",
				})
				return
			}
			http.Error(w, "Authentication failed: "+err.Error(), http.StatusUnauthorized)
			return
		}

		log.Printf("[API DEBUG] Bambu discover: Authentication successful, discovering printers...")

		// Discover printers
		printers, err := provider.DiscoverPrinters(ctx)
		if err != nil {
			log.Printf("[API DEBUG] Bambu discover: Discovery failed - %v", err)
			http.Error(w, "Discovery failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("[API DEBUG] Bambu discover: Found %d printers", len(printers))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"printers": printers,
		})
	})

	// Network Scanner - Discover Printers
	mux.HandleFunc("/api/scanner/discover", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API DEBUG] Network scanner discover request received")

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		printers, err := scanner.Discover(ctx)
		if err != nil {
			log.Printf("[API DEBUG] Network scanner: Scan failed - %v", err)
			http.Error(w, "Scan failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("[API DEBUG] Network scanner: Found %d printers", len(printers))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"printers": printers,
		})
	})

	// Bambu Bind Printer
	mux.HandleFunc("/api/bambu/bind", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Serial    string `json:"serial"`
			Name      string `json:"name"`
			Model     string `json:"model"`
			ClusterID int64  `json:"cluster_id"`
			IPAddress string `json:"ip_address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}

		// Get default Bambu account for credentials
		acc, err := database.GetDefaultBambuAccount()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if acc == nil {
			http.Error(w, "No Bambu account configured", http.StatusNotFound)
			return
		}

		// Add printer to database
		extra := map[string]string{
			"serial":      req.Serial,
			"region":      acc.Region,
			"access_code": "", // Will be set later if using LAN
			"local_ip":    req.IPAddress,
			"mode":        "CLOUD",
		}
		extraJSON, _ := json.Marshal(extra)

		err = database.AddPrinter(
			req.ClusterID,
			req.Name,
			req.IPAddress,
			"BAMBULAB",
			req.Serial,
			"", // access_code (empty for cloud mode)
			acc.Email,
			acc.PasswordHash,
			acc.AccessToken,
			"",
			string(extraJSON),
		)
		if err != nil {
			http.Error(w, "Failed to add printer: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Printer '%s' (%s) bound successfully", req.Name, req.Serial),
		})
	})

	// Bambu Account List
	mux.HandleFunc("/api/bambu/accounts", func(w http.ResponseWriter, r *http.Request) {
		accounts, err := database.ListBambuAccounts()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Mask sensitive data
		for i := range accounts {
			accounts[i].PasswordHash = "***REDACTED***"
			accounts[i].TFASecret = "***REDACTED***"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"accounts": accounts,
		})
	})

	// Bambu Delete Account
	mux.HandleFunc("/api/bambu/account/delete", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		if err := database.DeleteBambuAccount(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	})

	// --- 5. START SERVER ---
	// Serve static files from root dist/ folder with cache control
	mux.Handle("/", serveWithCacheControl(http.FileServer(http.Dir("dist"))))
	handlerWithCORS := middlewareCORS(mux)

	fmt.Printf("📡 Nexus Hub API started on port: %d\n", port)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), handlerWithCORS); err != nil {
		fmt.Printf("Critical server error: %v\n", err)
	}
}

func middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Force cache busting for ALL responses (prevents stale UI)
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		// CORS headers
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

// uploadToMoonraker uploads a G-Code file to a Moonraker printer
func uploadToMoonraker(printerIP, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}
	io.Copy(part, file)

	_ = writer.WriteField("print", "true")
	writer.Close()

	url := fmt.Sprintf("http://%s/server/files/upload", printerIP)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Moonraker error: %d - %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// serveWithCacheControl serves static files (cache control handled by middlewareCORS)
func serveWithCacheControl(fs http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Let middlewareCORS handle cache headers
		fs.ServeHTTP(w, r)
	})
}

// handleSlicerUpload handles file uploads from slicers (OctoPrint-compatible API)
func handleSlicerUpload(database *db.Database, hub *api.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[SLICER GRABBER] Received upload from slicer")

		// Ensure uploads directory exists
		uploadDir := "./uploads"
		thumbnailDir := filepath.Join(uploadDir, "thumbnails")
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
			return
		}
		if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
			http.Error(w, "Failed to create thumbnail directory", http.StatusInternalServerError)
			return
		}

		// Parse multipart form (100MB limit)
		if err := r.ParseMultipartForm(100 << 20); err != nil {
			http.Error(w, "File too large (max 100MB)", http.StatusBadRequest)
			return
		}

		// Get file from form
		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "No file provided", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Validate file extension
		ext := strings.ToLower(filepath.Ext(handler.Filename))
		if ext != ".gcode" && ext != ".g" && ext != ".3mf" && ext != ".gcode.3mf" {
			http.Error(w, "Invalid file type. Only .gcode and .3mf files are allowed", http.StatusBadRequest)
			return
		}

		// Save file
		dstPath := filepath.Join(uploadDir, handler.Filename)
		dst, err := os.Create(dstPath)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		log.Printf("[SLICER GRABBER] Saved file: %s (%d bytes)", handler.Filename, handler.Size)

		// Analyze file
		analysis, err := parser.AnalyzeFile(dstPath, uploadDir)
		if err != nil {
			log.Printf("[SLICER GRABBER] Analysis warning: %v", err)
			// Continue even if analysis fails
			analysis = &parser.AnalysisResult{
				Filename: handler.Filename,
				FileSize: handler.Size,
			}
		}

		log.Printf("[SLICER GRABBER] Analysis: time=%ds, filament=%.1fg, thumbnail=%s",
			analysis.EstimatedTimeS, analysis.FilamentWeight, analysis.ThumbnailURL)

		// Create print job
		if err := database.AddPrintJob(handler.Filename, analysis.FilamentType, "Unknown", uint64(analysis.EstimatedTimeS)); err != nil {
			http.Error(w, "Failed to create print job", http.StatusInternalServerError)
			return
		}

		// Update job with metadata
		if analysis.ThumbnailURL != "" || analysis.FilamentWeight > 0 {
			// Get the last job ID (simplified - should use returned ID)
			jobID, _, _, _ := database.GetNextPendingJob()
			if jobID > 0 {
				database.UpdateJobMetadata(jobID, analysis.ThumbnailURL, analysis.FilamentWeight)
			}
		}

		// Log to system
		hub.Send(api.NexusPacket{
			EventType: "SYSTEM_LOG",
			Log: &api.LogPayload{
				Module:    "SLICER",
				Level:     "INFO",
				Message:   fmt.Sprintf("Slicer uploaded: %s (time: %ds, filament: %.1fg)", handler.Filename, analysis.EstimatedTimeS, analysis.FilamentWeight),
				Timestamp: time.Now(),
			},
		})

		// Return OctoPrint-compatible response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"done": true,
			"files": map[string]interface{}{
				"local": map[string]interface{}{
					"name":      handler.Filename,
					"path":      handler.Filename,
					"display":   handler.Filename,
					"origin":    "local",
					"type":      ext,
					"size":      handler.Size,
					"date":      time.Now().Unix(),
					"thumbnail": analysis.ThumbnailURL,
					"printTime": analysis.EstimatedTimeS,
					"filament":  analysis.FilamentWeight,
				},
			},
		})
	}
}

// sendBambuCommand sends a command to a Bambu printer via MQTT
// Uses exact payloads from Python reverse-engineering
func sendBambuCommand(printer db.Printer, action string, hub *api.Hub) {
	// Map action to correct Bambu JSON payload (exact from Python lib)
	var payload string

	switch action {
	case "pause":
		// NO sequence_id for system commands (from Python lib)
		payload = `{"print": {"command": "pause"}}`
	case "resume":
		payload = `{"print": {"command": "resume"}}`
	case "stop":
		payload = `{"print": {"command": "stop"}}`
	case "g28", "home":
		// sequence_id "0" for G-code commands (from Python lib)
		payload = `{"print": {"sequence_id": "0", "command": "gcode_line", "param": "G28 \n"}}`
	default:
		hub.Send(api.NexusPacket{
			EventType: "SYSTEM_LOG",
			Log: &api.LogPayload{
				Module:    "WEB",
				Level:     "WARN",
				Message:   fmt.Sprintf("Unknown command '%s' for Bambu %s", action, printer.Name),
				Timestamp: time.Now(),
			},
		})
		return
	}

	// Create MQTT client for this command
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcps://%s:8883", printer.IPAddress))
	opts.SetClientID(fmt.Sprintf("nexus-hub-cmd-%s", printer.Serial))
	opts.SetUsername("bblp")
	opts.SetPassword(printer.AccessCode)
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

	client := mqtt.NewClient(opts)

	log.Printf("[WEB] Sending Bambu command to %s: %s", printer.Name, payload)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("[WEB] Failed to connect to Bambu %s: %v", printer.Name, token.Error())
		hub.Send(api.NexusPacket{
			EventType: "SYSTEM_LOG",
			Log: &api.LogPayload{
				Module:    "WEB",
				Level:     "ERROR",
				Message:   fmt.Sprintf("Failed to connect to Bambu %s: %v", printer.Name, token.Error()),
				Timestamp: time.Now(),
			},
		})
		return
	}
	defer client.Disconnect(250)

	// Send command to request topic
	requestTopic := fmt.Sprintf("device/%s/request", printer.Serial)

	if token := client.Publish(requestTopic, 1, false, payload); token.Wait() && token.Error() != nil {
		log.Printf("[WEB] Failed to send command to Bambu %s: %v", printer.Name, token.Error())
		hub.Send(api.NexusPacket{
			EventType: "SYSTEM_LOG",
			Log: &api.LogPayload{
				Module:    "WEB",
				Level:     "ERROR",
				Message:   fmt.Sprintf("Failed to send command '%s' to Bambu %s: %v", action, printer.Name, token.Error()),
				Timestamp: time.Now(),
			},
		})
	} else {
		log.Printf("[WEB] Command sent successfully to Bambu %s", printer.Name)
		hub.Send(api.NexusPacket{
			EventType: "SYSTEM_LOG",
			Log: &api.LogPayload{
				Module:    "WEB",
				Level:     "INFO",
				Message:   fmt.Sprintf("Sent command '%s' to Bambu %s", action, printer.Name),
				Timestamp: time.Now(),
			},
		})
	}
}
