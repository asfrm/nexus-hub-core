package main

import (
	"context"
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
	log.Println("🚀 Starting Nexus Hub Emulation Tester")

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	log.Println("✅ Testing Complete. Shutting down...")
	time.Sleep(1 * time.Second)
}

func runTestScenario(database *db.Database, hub *api.Hub) {
	log.Println("🧪 Running Test Scenario...")

	// Step A: Create Cluster
	log.Println("   [A] Creating Cluster...")
	if err := database.CreateCluster("Test Cluster", "X1C"); err != nil {
		log.Printf("❌ Failed to create cluster: %v", err)
		return
	}

	// Step B: Add Printers
	log.Println("   [B] Adding Mock Printers...")
	// Moonraker
	if err := database.AddPrinter(1, "Mock Moonraker", "127.0.0.1:7125", "MOONRAKER", "MOON001", "", "", "", "", "", ""); err != nil {
		log.Printf("❌ Failed to add Moonraker: %v", err)
	}
	// Bambu (using local IP 127.0.0.1 for mock)
	if err := database.AddPrinter(1, "Mock Bambu", "127.0.0.1", "BAMBULAB", "BAMB001", "12345678", "bblp", "", "", "", `{"local_ip": "127.0.0.1"}`); err != nil {
		log.Printf("❌ Failed to add Bambu: %v", err)
	}

	// Step C: Create a Job
	log.Println("   [C] Creating Print Job...")
	if err := database.AddPrintJob("test_part.gcode", "PLA", "X1C", 3600); err != nil {
		log.Printf("❌ Failed to add print job: %v", err)
	}

	// Step D: Trigger Assignment (Manual first)
	log.Println("   [D] Manually Assigning Job...")
	if err := database.AssignJob(1, 1, 1); err != nil {
		log.Printf("❌ Failed to assign job: %v", err)
	}

	// Step E: Check Status
	time.Sleep(2 * time.Second)
	job, _ := database.GetJobByID(1)
	log.Printf("   [E] Job Status: %s (Target: %d)", job.Status, job.TargetPrinterID)

	// Step F: Verify Poller activity
	log.Println("   [F] Waiting for Poller heartbeat...")
	time.Sleep(5 * time.Second)
	
	stats, _ := database.GetPrinterWearStats()
	for _, s := range stats {
		log.Printf("   [F] Printer %s Wear Score: %.2f", s.PrinterName, s.WearScore)
	}
}
