package slicer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func UploadGCodeHandler(database *db.Database, hub *api.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uploadDir := "./uploads"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
			return
		}

		// Read multipart form with 100MB limit
		r.ParseMultipartForm(100 << 20)
		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Upload error", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Validate file extension
		ext := strings.ToLower(filepath.Ext(handler.Filename))
		allowedExts := map[string]bool{
			".gcode":     true,
			".g":         true,
			".3mf":       true,
			".gcode.3mf": true,
		}

		if !allowedExts[ext] && !strings.HasSuffix(strings.ToLower(handler.Filename), ".gcode.3mf") {
			http.Error(w, "Invalid file type. Supported: .gcode, .3mf (sliced only)", http.StatusBadRequest)
			return
		}
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

		// Parse and save metadata
		meta := ParseGCodeMetadata(dstPath)
		if err := database.AddPrintJob(handler.Filename, meta.FilamentType, meta.PrinterModel, meta.EstimatedTimeS); err != nil {
			http.Error(w, "Failed to create print job", http.StatusInternalServerError)
			return
		}

		// Log to system
		hub.Send(api.NexusPacket{
			EventType: "SYSTEM_LOG",
			Log: &api.LogPayload{
				Module:    "SLICER",
				Level:     "INFO",
				Message:   fmt.Sprintf("Task '%s' accepted for cluster: [%s] (Filament: %s)", handler.Filename, meta.PrinterModel, meta.FilamentType),
				Timestamp: time.Now(),
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": map[string]interface{}{
				"action": "upload",
				"item": map[string]string{
					"path": handler.Filename,
				},
			},
		})
	}
}
