package slicer

import (
	"fmt"
	"io"
	"net/http"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"os"
	"path/filepath"
	"time"
)

func UploadGCodeHandler(database *db.Database, tx chan api.NexusPacket) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uploadDir := "./gcode_vault"
		os.MkdirAll(uploadDir, 0755)

		// Читаем multipart форму
		r.ParseMultipartForm(100 << 20) // 100MB лимит
		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Upload error", http.StatusBadRequest)
			return
		}
		defer file.Close()

		dstPath := filepath.Join(uploadDir, handler.Filename)
		dst, _ := os.Create(dstPath)
		defer dst.Close()
		io.Copy(dst, file)

		// Парсим и сохраняем
		meta := ParseGCodeMetadata(dstPath)
		_ = database.AddPrintJob(handler.Filename, meta.FilamentType, meta.PrinterModel, meta.EstimatedTimeS)

		// Лог в систему (теперь с указанием кластера!)
		tx <- api.NexusPacket{
			EventType: "SYSTEM_LOG",
			Log: &api.LogPayload{
				Module:    "SLICER",
				Level:     "INFO",
				Message:   fmt.Sprintf("Задача '%s' принята для кластера: [%s] (Пластик: %s)", handler.Filename, meta.PrinterModel, meta.FilamentType),
				Timestamp: time.Now(),
			},
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"result": {"action": "upload", "item": {"path": "%s"}}}`, handler.Filename)
	}
}
