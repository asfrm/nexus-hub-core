package printer

import (
	"context"
	"fmt"
	"log"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"nexus-hub-core/printer/bambu"
	"sync"
	"time"
)

// Poller manages periodic status checks for all printers
type Poller struct {
	database     *db.Database
	hub          *api.Hub
	interval     time.Duration
	mu           sync.Mutex
	stopCh       chan struct{}
	bambuClients map[int64]*bambu.Client
}

// NewPoller creates a new printer status poller
func NewPoller(database *db.Database, hub *api.Hub) *Poller {
	return &Poller{
		database:     database,
		hub:          hub,
		interval:     5 * time.Second,
		stopCh:       make(chan struct{}),
		bambuClients: make(map[int64]*bambu.Client),
	}
}

// Start begins the polling loop
func (p *Poller) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Printer poller shutting down...")
			return
		case <-ticker.C:
			p.pollAllPrinters(ctx)
		}
	}
}

func (p *Poller) pollAllPrinters(ctx context.Context) {
	clusters, err := p.database.GetClustersWithPrinters()
	if err != nil {
		log.Printf("Failed to get clusters for polling: %v", err)
		return
	}

	for _, cluster := range clusters {
		if !cluster.IsEnabled {
			continue
		}

		for _, printer := range cluster.Printers {
			go p.pollPrinter(ctx, printer)
		}
	}
}

func (p *Poller) pollPrinter(ctx context.Context, printer db.Printer) {
	if printer.PrinterType == "BAMBULAB" {
		p.pollBambuPrinter(ctx, printer)
		return
	}

	// For Moonraker printers (implementation pending)
	log.Printf("[POLLER] Moonraker polling not yet implemented for %s", printer.Name)
}

func (p *Poller) pollBambuPrinter(ctx context.Context, printer db.Printer) {
	p.mu.Lock()
	client, exists := p.bambuClients[printer.ID]
	if !exists {
		// Create new Bambu client
		client = bambu.NewClient()

		// Connect to printer
		err := client.ConnectLAN(printer.IPAddress, printer.AccessCode, printer.Serial)
		if err != nil {
			p.mu.Unlock()
			log.Printf("[BAMBU POLLER] Failed to connect to %s: %v", printer.Name, err)
			return
		}

		p.bambuClients[printer.ID] = client

		// Start status listener goroutine
		go func(id int64, name string, c *bambu.Client) {
			for status := range c.GetStatusChan() {
				p.hub.Send(api.NexusPacket{
					EventType: "PRINTER_STATUS",
					Payload: map[string]interface{}{
						"printer_id":           id,
						"printer_name":         name,
						"printer_type":         "BAMBULAB",
						"status":               status.State,
						"filename":             status.Filename,
						"progress":             status.Progress,
						"time_left":            status.TimeLeft,
						"layer":                status.Layer,
						"total_layers":         status.TotalLayers,
						"nozzle_temper":        status.Temperatures["nozzle"],
						"bed_temper":           status.Temperatures["bed"],
						"nozzle_target_temper": status.Targets["nozzle"],
						"bed_target_temper":    status.Targets["bed"],
						"fan_speed":            status.FanSpeed,
						"speed_level":          status.SpeedLevel,
					},
				})
			}
		}(printer.ID, printer.Name, client)
	}
	p.mu.Unlock()

	// Get current status (for logging)
	status := client.GetStatus()
	log.Printf("[BAMBU POLLER] %s: %s | Progress: %.1f%% | Nozzle: %.1f/%.1f°C | Bed: %.1f/%.1f°C",
		printer.Name, status.State, status.Progress,
		status.Temperatures["nozzle"], status.Targets["nozzle"],
		status.Temperatures["bed"], status.Targets["bed"])
}

func (p *Poller) markJobComplete(printerID int64) {
	// Find the active job for this printer
	jobs, err := p.database.GetJobList()
	if err != nil {
		log.Printf("Failed to get jobs for completion check: %v", err)
		return
	}

	for _, job := range jobs {
		if job.TargetPrinterID == printerID && job.Status == "PRINTING" {
			p.database.UpdateJobStatus(job.ID, "COMPLETED")
			p.hub.Send(api.NexusPacket{
				EventType: "SYSTEM_LOG",
				Log: &api.LogPayload{
					Module:    "POLLER",
					Level:     "INFO",
					Message:   fmt.Sprintf("Job '%s' completed successfully", job.Filename),
					Timestamp: time.Now(),
				},
			})
			break
		}
	}
}
