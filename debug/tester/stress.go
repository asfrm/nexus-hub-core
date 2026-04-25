package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"sync"
	"time"
)

func RunMegaBlitz(ctx context.Context, baseURL string) {
	log.Println("⚡ INITIALIZING MEGA BLITZ ATTACK (30s duration)...")
	
	blitzCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// 1. API Flood - 5000 Workers
	log.Println("   [1] Launching 5000 API Flood Workers...")
	for i := 0; i < 5000; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client := &http.Client{Timeout: 2 * time.Second}
			for {
				select {
				case <-blitzCtx.Done():
					return
				default:
					resp, err := client.Get(baseURL + "/api/stats")
					if err == nil {
						resp.Body.Close()
					}
				}
			}
		}(i)
	}

	// 2. Control Command Storm
	log.Println("   [2] Launching Printer Control Storm...")
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client := &http.Client{Timeout: 2 * time.Second}
			for {
				select {
				case <-blitzCtx.Done():
					return
				default:
					printerID := (id % 1000) + 1
					url := fmt.Sprintf("%s/api/printers/control?id=%d&action=pause", baseURL, printerID)
					resp, err := client.Get(url)
					if err == nil {
						resp.Body.Close()
					}
				}
			}
		}(i)
	}

	// 3. Massive Upload Burst
	log.Println("   [3] Launching 50 Concurrent 50MB Uploads...")
	data := make([]byte, 50*1024*1024)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, _ := writer.CreateFormFile("file", fmt.Sprintf("blitz_%d.3mf", id))
			part.Write(data)
			writer.Close()

			req, _ := http.NewRequestWithContext(blitzCtx, "POST", baseURL+"/api/files/local", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			
			client := &http.Client{}
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}(i)
	}

	log.Println("🚀 BLITZ IN PROGRESS...")
	wg.Wait()
	log.Println("🏁 BLITZ COMPLETED.")
}

func RunStressTest(baseURL string) {
	// Replaced by MegaBlitz
}

func RunBreakerTest(baseURL string) {
	// Replaced by MegaBlitz
}

func RunRealLifeStressTest(baseURL string) {
	// Replaced by MegaBlitz
}
