package main

import (
	"context"
	"flag"
	"log"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"nexus-hub-core/dispatcher"
	"nexus-hub-core/printer"
	"nexus-hub-core/scheduler"
	"nexus-hub-core/web"
	"os"
	"os/signal"
	"syscall"
)

// Version information injected by CI/CD via ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	// 0. Version info
	log.Printf("Nexus Hub Core v%s (commit: %s, built: %s)", Version, GitCommit, BuildDate)

	// 1. Command line flags
	isDebug := flag.Bool("debug", false, "enable debug logging")
	versionFlag := flag.Bool("version", false, "show version information")
	flag.Parse()

	// Show version and exit
	if *versionFlag {
		log.Printf("Nexus Hub Core %s", Version)
		log.Printf("Git Commit: %s", GitCommit)
		log.Printf("Build Date: %s", BuildDate)
		os.Exit(0)
	}

	// 2. Initialize database (SQLite)
	database, err := db.NewDatabase("hub.db")
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	// 3. Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 4. Channels with pub-sub hub for broadcasting
	hub := api.NewHub()
	hub.Run(ctx)

	// 5. Worker for logging and distribution
	go func() {
		for packet := range hub.GetInternalChan() {
			// Save to database
			if err := database.SaveEvent(packet); err != nil {
				log.Printf("Failed to save event: %v", err)
			}
		}
	}()

	// 6. Start modules (Goroutines)
	if *isDebug {
		go debugLogger(ctx, hub.Subscribe())
	}

	// Start printer status poller
	poller := printer.NewPoller(database, hub)
	go poller.Start(ctx)

	// Start job scheduler
	sched := scheduler.NewScheduler(database, hub, &dispatcher.AutoMode)
	go sched.Start(ctx)

	// Start dispatcher (manual mode support)
	go dispatcher.StartDispatching(ctx, database, hub)

	// Start web server (API + debug.html only)
	go web.StartWebEngine(database, hub, 8080)

	// 7. Wait for shutdown signal (Ctrl+C)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down Nexus Hub...")
	cancel()
}

func debugLogger(ctx context.Context, ch <-chan api.NexusPacket) {
	for {
		select {
		case <-ctx.Done():
			return
		case packet, ok := <-ch:
			if !ok {
				return
			}
			log.Printf("[DEBUG] Received packet: %s", packet.EventType)
		}
	}
}
