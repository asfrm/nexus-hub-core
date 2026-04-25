package main

import (
	"bytes"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"sync"
	"time"
)

func RunStressTest(baseURL string) {
	log.Println("🔥 Starting Stress & Concurrency Testing...")
	
	var wg sync.WaitGroup
	
	// 1. Concurrent GET Requests (WAL Verification)
	log.Println("   [1] Stressing GET endpoints under load...")
	endpoints := []string{"/api/stats", "/api/analytics", "/api/clusters", "/api/jobs/list"}
	for _, ep := range endpoints {
		wg.Add(1)
		go func(endpoint string) {
			defer wg.Done()
			start := time.Now()
			successCount := 0
			for i := 0; i < 500; i++ {
				resp, err := http.Get(baseURL + endpoint)
				if err == nil && resp.StatusCode == http.StatusOK {
					successCount++
					resp.Body.Close()
				}
			}
			log.Printf("       - %s: %d/500 OK (Avg: %v)", endpoint, successCount, time.Since(start)/500)
		}(ep)
	}

	// 2. 50MB Payload Test
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("   [2] Testing 50MB large file upload...")
		data := make([]byte, 50*1024*1024) // 50MB
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "large_model.3mf")
		part.Write(data)
		writer.Close()

		resp, err := http.Post(baseURL+"/api/files/local", writer.FormDataContentType(), body)
		if err != nil {
			log.Printf("       ❌ 50MB Upload failed: %v", err)
			return
		}
		log.Printf("       ✅ 50MB Upload response: %d", resp.StatusCode)
		resp.Body.Close()
	}()

	wg.Wait()
}

func RunRealLifeStressTest(baseURL string) {
	log.Println("🌍 Starting REAL LIFE Scenario Test (1000 Printers + 1000 Active Users)...")
	
	var wg sync.WaitGroup

	// 1. Simulate 1000 Users Watching Live Updates (WS/GET)
	log.Println("   [1] Spawning 1000 concurrent user sessions...")
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Each user checks stats every few seconds
			for j := 0; j < 5; j++ {
				resp, err := http.Get(baseURL + "/api/stats")
				if err == nil {
					resp.Body.Close()
				}
				time.Sleep(200 * time.Millisecond)
			}
		}(i)
	}

	// 2. Simulate User Actions (Button Presses)
	log.Println("   [2] Simulating 500 concurrent printer control commands...")
	for i := 1; i <= 500; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Randomly home or pause
			action := "g28"
			if id % 2 == 0 { action = "pause" }
			
			url := fmt.Sprintf("%s/api/printers/control?id=%d&action=%s", baseURL, id, action)
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
			}
		}(i)
	}

	// 3. Concurrent File Uploads (Multiple Users Slicing)
	log.Println("   [3] Bursting 10 concurrent 5MB uploads...")
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			data := make([]byte, 5*1024*1024)
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, _ := writer.CreateFormFile("file", fmt.Sprintf("user_%d_print.gcode", id))
			part.Write(data)
			writer.Close()

			resp, err := http.Post(baseURL+"/api/files/local", writer.FormDataContentType(), body)
			if err == nil {
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()
	log.Println("   ✅ Real-life scenario complete.")
}

func RunBreakerTest(baseURL string) {
	log.Println("🔨 Starting CRASH TEST (Absolute Limits)...")
	
	// Flood with extremely high concurrency
	concurrency := 1500
	log.Printf("   [*] Spawning %d concurrent attackers...", concurrency)
	
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			for {
				// Rapid fire small requests
				resp, err := http.Get(baseURL + "/api/health")
				if err == nil {
					resp.Body.Close()
				}
				
				// Occasional heavy write
				if id % 100 == 0 {
					data := []byte(`{"name": "Attacker Cluster", "target_model": "Exploit"}`)
					http.Post(baseURL+"/api/clusters/create", "application/json", bytes.NewBuffer(data))
				}
				
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	// Keep main thread alive to watch it burn
	log.Println("   [*] Attack in progress. System should hit file descriptor limits or memory pressure...")
	start := time.Now()
	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Second)
		resp, err := http.Get(baseURL + "/api/health")
		if err != nil {
			log.Printf("   💥 SYSTEM CRASHED after %v! Error: %v", time.Since(start), err)
			return
		}
		resp.Body.Close()
		log.Printf("   ... %d seconds still alive", i+1)
	}
	log.Println("   🏁 System survived 15s of extreme flood.")
}
