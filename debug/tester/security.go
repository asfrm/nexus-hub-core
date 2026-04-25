package main

import (
	"log"
	"net/http"
	"net/url"
)

func RunSecurityTest(baseURL string) {
	log.Println("🛡️ Starting Basic Security & Penetration Testing...")

	// 1. SQL Injection Attempt on Cluster Delete
	log.Println("   [1] Testing SQL injection on /api/clusters/delete...")
	vulnParams := []string{"1 OR 1=1", "' OR '1'='1", "1; DROP TABLE users;"}
	for _, p := range vulnParams {
		resp, err := http.Get(baseURL + "/api/clusters/delete?id=" + url.QueryEscape(p))
		if err == nil {
			if resp.StatusCode == http.StatusInternalServerError {
				log.Printf("       ⚠️ Potential vulnerability detected with param: %s (Status 500)", p)
			} else {
				log.Printf("       ✅ Handled param: %s (Status %d)", p, resp.StatusCode)
			}
			resp.Body.Close()
		}
	}

	// 2. Path Traversal Attempt
	log.Println("   [2] Testing Path Traversal on static file handler...")
	traversalPaths := []string{"/../../etc/passwd", "/debug.html/../../go.mod", "/uploads/../main.go"}
	for _, p := range traversalPaths {
		resp, err := http.Get(baseURL + p)
		if err == nil {
			if resp.StatusCode == http.StatusOK && resp.ContentLength > 0 {
				log.Printf("       🚨 VULNERABILITY: Path Traversal succeeded for: %s", p)
			} else {
				log.Printf("       ✅ Blocked path: %s (Status %d)", p, resp.StatusCode)
			}
			resp.Body.Close()
		}
	}
}
