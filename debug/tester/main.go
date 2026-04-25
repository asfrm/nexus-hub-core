package main

import (
	"context"
	"fmt"
	"log"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"nexus-hub-core/dispatcher"
	"nexus-hub-core/printer"
	"nexus-hub-core/scheduler"
	"nexus-hub-core/web"
	"os"
	"time"
)

func main() {
	log.Println("🚀 Starting Nexus Hub MEGA BLITZ Tester")

	// Global Timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 1. Start Mocks
	StartMoonrakerMock(7125)
	StartBambuMock("127.0.0.1", 1883)

	// 2. Setup Temporary Environment
	tmpDbPath := "test_hub.db"
	os.Remove(tmpDbPath)
	defer os.Remove(tmpDbPath)

	database, err := db.NewDatabase(tmpDbPath)
	if err != nil {
		log.Fatalf("Failed to create test DB: %v", err)
	}

	hub := api.NewHub()

	// 3. Initialize Hub Components
	log.Println("📦 Initializing Hub Components...")

	poller := printer.NewPoller(database, hub)
	go poller.Start(ctx)

	sched := scheduler.NewScheduler(database, hub, &dispatcher.AutoMode)
	go sched.Start(ctx)

	// Start web engine on a test port
	go web.StartWebEngine(database, hub, 8081)

	time.Sleep(2 * time.Second) // Wait for startup

	// 4. Run Test Scenario
	runTestScenario(database, hub)

	// 5. MEGA BLITZ Stress & Security Tests
	testURL := "http://localhost:8081"
	RunSecurityTest(testURL)
	RunMegaBlitz(ctx, testURL)

	log.Println("✅ MEGA BLITZ Complete. Shutting down...")
	time.Sleep(1 * time.Second)
}

func runTestScenario(database *db.Database, hub *api.Hub) {
	log.Println("🧪 Pre-populating 1000 printers via Transaction...")

	tx, _ := database.Conn.Begin()
	for i := 1; i <= 500; i++ {
		// Moonraker
		tx.Exec("INSERT INTO printers (cluster_id, name, ip_address, printer_type, serial, access_code, username, password, access_token, api_key, extra) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			1, fmt.Sprintf("Moon-%d", i), "127.0.0.1:7125", "MOONRAKER", fmt.Sprintf("MOON%03d", i), "", "", "", "", "", "")

		// Bambu
		tx.Exec("INSERT INTO printers (cluster_id, name, ip_address, printer_type, serial, access_code, username, password, access_token, api_key, extra) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			1, fmt.Sprintf("Bambu-%d", i), "127.0.0.1", "BAMBULAB", fmt.Sprintf("BAMB%03d", i), "12345678", "bblp", "", "", "", `{"local_ip": "127.0.0.1"}`)
	}
	tx.Commit()
	log.Println("   ✅ Database populated.")
}
