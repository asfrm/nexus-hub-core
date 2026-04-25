package printer

import (
	"context"
	"fmt"
	"log"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"nexus-hub-core/printer/bambu"
	"nexus-hub-core/printer/moonraker"
	"nexus-hub-core/printer/types"
	"strings"
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
	} else if printer.PrinterType == "MOONRAKER" {
		p.pollMoonrakerPrinter(ctx, printer)
	}
}

func (p *Poller) pollMoonrakerPrinter(ctx context.Context, printer db.Printer) {
	status, err := moonraker.GetStatus(printer.IPAddress)
	if err != nil {
		log.Printf("[MOONRAKER POLLER] Failed to get status for %s: %v", printer.Name, err)
		return
	}

	state := strings.ToUpper(status.State)
	if state == "COMPLETE" || state == "FINISHED" {
		p.markJobComplete(printer.ID)
	}

	p.hub.Send(api.NexusPacket{
		EventType: "PRINTER_STATUS",
		Payload: map[string]interface{}{
			"printer_id":    printer.ID,
			"printer_name":  printer.Name,
			"printer_type":  "MOONRAKER",
			"status":        state,
			"filename":      status.Filename,
			"progress":      status.Progress,
			"time_left":     status.TimeLeft,
			"nozzle_temper": status.TempNozzle,
			"bed_temper":    status.TempBed,
		},
	})
}

func (p *Poller) pollBambuPrinter(ctx context.Context, printer db.Printer) {
	p.mu.Lock()
	provider, exists := p.bambuProviders[printer.ID]
	if !exists {
		// Start connection in a goroutine so it doesn't block the poller
		go func(pr db.Printer) {
			config := types.PrinterConfig{
				PrinterID: pr.ID,
				Address:   pr.IPAddress,
				Extra: map[string]string{
					"local_ip":    pr.IPAddress,
					"access_code": pr.AccessCode,
					"serial":      pr.Serial,
				},
			}
			prov := bambu.NewBambuProvider(config)

			// Use a separate context with timeout for connection
			connCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := prov.Connect(connCtx); err != nil {
				log.Printf("[BAMBU] Failed to connect to %s: %v", pr.Name, err)
				return
			}

			p.mu.Lock()
			p.bambuProviders[pr.ID] = prov
			p.mu.Unlock()

			// Start status listener
			sdkPrinter := prov.GetPrinter()
			updateChan := sdkPrinter.GetUpdateChannel()

			for range updateChan {
				s, err := prov.GetStatus(ctx)
				if err != nil {
					continue
				}

				if s.State == "COMPLETE" {
					p.markJobComplete(pr.ID)
				}

				p.hub.Send(api.NexusPacket{
					EventType: "PRINTER_STATUS",
					Payload: map[string]interface{}{
						"printer_id":    pr.ID,
						"printer_name":  pr.Name,
						"printer_type":  "BAMBULAB",
						"status":        s.State,
						"filename":      s.Filename,
						"progress":      s.Progress,
						"time_left":     s.TimeLeft,
						"nozzle_temper": s.Temperatures["nozzle"],
						"bed_temper":    s.Temperatures["bed"],
						"ams_data":      s.VendorData["ams"],
					},
				})
			}
		}(printer)
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	// Get status for logging if already connected
	status, err := provider.GetStatus(ctx)
	if err == nil {
		log.Printf("[BAMBU POLLER] %s: %s | Progress: %.1f%%", printer.Name, status.State, status.Progress)
	}
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
