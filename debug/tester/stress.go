package main

import (
	"bytes"
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

func RunBreakerTest(baseURL string) {
	log.Println("🔨 Starting CRASH TEST (Absolute Limits)...")
	
	// Flood with extremely high concurrency
	concurrency := 1000
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
