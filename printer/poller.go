package printer

import (
	"context"
	"fmt"
	"log"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"nexus-hub-core/printer/bambu"
	"nexus-hub-core/printer/types"
	"sync"
	"time"
)

// Poller manages periodic status checks for all printers
type Poller struct {
	database       *db.Database
	hub            *api.Hub
	interval       time.Duration
	mu             sync.Mutex
	stopCh         chan struct{}
	bambuProviders map[int64]*bambu.BambuProvider
}

// NewPoller creates a new printer status poller
func NewPoller(database *db.Database, hub *api.Hub) *Poller {
	return &Poller{
		database:       database,
		hub:            hub,
		interval:       5 * time.Second,
		stopCh:         make(chan struct{}),
		bambuProviders: make(map[int64]*bambu.BambuProvider),
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
	provider, exists := p.bambuProviders[printer.ID]
	if !exists {
		// Create new Bambu provider
		config := types.PrinterConfig{
			PrinterID: printer.ID,
			Address:   printer.IPAddress,
			Extra: map[string]string{
				"local_ip":    printer.IPAddress,
				"access_code": printer.AccessCode,
				"serial":      printer.Serial,
			},
		}
		provider = bambu.NewBambuProvider(config)

		// Connect to printer
		if err := provider.Connect(ctx); err != nil {
			p.mu.Unlock()
			log.Printf("[BAMBU] Failed to connect to %s: %v", printer.Name, err)
			return
		}

		p.bambuProviders[printer.ID] = provider

		// Start status listener goroutine using SDK's update channel
		go func(id int64, name string, prov *bambu.BambuProvider) {
			sdkPrinter := prov.GetPrinter()
			updateChan := sdkPrinter.GetUpdateChannel()

			for range updateChan {
				status, err := prov.GetStatus(ctx)
				if err != nil {
					log.Printf("[BAMBU POLLER] Failed to get status for %s: %v", name, err)
					continue
				}

				// --- ENSURE COMPLETE TRIGGER FIRES ---
				if status.State == "COMPLETE" {
					p.markJobComplete(id)
				}

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
						"ams_data":             status.VendorData["ams"], // --- NEW: Broadcast AMS data
					},
				})
			}
		}(printer.ID, printer.Name, provider)
	}
	p.mu.Unlock()

	// Get current status (for logging)
	status, err := provider.GetStatus(context.Background())
	if err != nil {
		log.Printf("[BAMBU POLLER] Failed to get status for %s: %v", printer.Name, err)
		return
	}

	log.Printf("[BAMBU POLLER] %s: %s | Progress: %.1f%% | Nozzle: %.1f°C | Bed: %.1f°C",
		printer.Name, status.State, status.Progress,
		status.Temperatures["nozzle"], status.Temperatures["bed"])
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
