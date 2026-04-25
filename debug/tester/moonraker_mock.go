package main

import (
	"fmt"
	"log"
	"net/http"
)

type MoonrakerMock struct {
	Port int
}

func StartMoonrakerMock(port int) *MoonrakerMock {
	m := &MoonrakerMock{Port: port}
	mux := http.NewServeMux()

	mux.HandleFunc("/server/files/upload", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[Moonraker Mock :%d] Received file upload", port)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"result": "ok"}`)
	})

	mux.HandleFunc("/printer/print/pause", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[Moonraker Mock :%d] Received pause", port)
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/printer/print/resume", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[Moonraker Mock :%d] Received resume", port)
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/printer/print/cancel", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[Moonraker Mock :%d] Received cancel", port)
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/printer/emergency_stop", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[Moonraker Mock :%d] Received estop", port)
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/printer/objects/query", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"result": {
				"print_stats": {
					"state": "printing",
					"filename": "test_part.gcode",
					"progress": 0.42,
					"print_time": 100.0,
					"total_duration": 240.0
				},
				"display_status": {
					"progress": 0.42
				},
				"heater_bed": {
					"temperature": 60.0,
					"target": 60.0
				}
			}
		}`)
	})

	go func() {
		log.Printf("[Moonraker Mock] Starting on port %d", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
			log.Printf("[Moonraker Mock] Error: %v", err)
		}
	}()

	return m
}
