package printer

import (
	"context"
	"fmt"
	"log"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"sync"
	"time"
)

// Poller manages periodic status checks for all printers
type Poller struct {
	database *db.Database
	hub      *api.Hub
	interval time.Duration
	mu       sync.Mutex
	stopCh   chan struct{}

	// Printer provider cache
	providers map[int64]PrinterProvider
}

// NewPoller creates a new printer status poller
func NewPoller(database *db.Database, hub *api.Hub) *Poller {
	return &Poller{
		database:  database,
		hub:       hub,
		interval:  5 * time.Second,
		stopCh:    make(chan struct{}),
		providers: make(map[int64]PrinterProvider),
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
	provider := p.getOrCreateProvider(printer)
	if provider == nil {
		return
	}

	status, err := provider.GetStatus(ctx)
	if err != nil {
		// Printer unreachable - mark as error
		if printer.Status != "ERROR" {
			p.database.UpdatePrinterStatus(printer.ID, "ERROR")
			p.hub.Send(api.NexusPacket{
				EventType: "SYSTEM_LOG",
				Log: &api.LogPayload{
					Module:    "POLLER",
					Level:     "ERROR",
					Message:   fmt.Sprintf("Printer %s (%s) unreachable: %v", printer.Name, printer.PrinterType, err),
					Timestamp: time.Now(),
				},
			})
		}
		return
	}

	// Determine new status
	newStatus := "IDLE"
	switch status.State {
	case "printing":
		newStatus = "PRINTING"
	case "paused":
		newStatus = "PAUSED"
	case "complete":
		newStatus = "IDLE"
	case "error":
		newStatus = "ERROR"
	}

	// Update database if status changed
	if newStatus != printer.Status {
		p.database.UpdatePrinterStatus(printer.ID, newStatus)

		// If print completed, update job status
		if newStatus == "IDLE" && printer.Status == "PRINTING" {
			p.markJobComplete(printer.ID)
		}

		p.hub.Send(api.NexusPacket{
			EventType: "SYSTEM_LOG",
			Log: &api.LogPayload{
				Module:    "POLLER",
				Level:     "INFO",
				Message:   fmt.Sprintf("Printer %s status changed: %s -> %s", printer.Name, printer.Status, newStatus),
				Timestamp: time.Now(),
			},
		})
	}

	// Broadcast printer status update
	p.hub.Send(api.NexusPacket{
		EventType: "PRINTER_STATUS",
		Payload: map[string]interface{}{
			"printer_id":   printer.ID,
			"printer_name": printer.Name,
			"printer_type": printer.PrinterType,
			"status":       newStatus,
			"filename":     status.Filename,
			"progress":     status.Progress,
			"time_left":    status.TimeLeft,
			"temperatures": status.Temperatures,
			"targets":      status.Targets,
			"fan_speed":    status.FanSpeed,
		},
	})
}

func (p *Poller) getOrCreateProvider(printer db.Printer) PrinterProvider {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check cache
	if provider, ok := p.providers[printer.ID]; ok {
		return provider
	}

	// Create provider based on printer type
	var provider PrinterProvider

	config := PrinterConfig{
		PrinterID:   printer.ID,
		Address:     printer.IPAddress,
		Username:    printer.Username,
		Password:    printer.Password,
		AccessToken: printer.AccessToken,
		Secret:      printer.Secret,
		Extra:       make(map[string]string),
	}

	// Parse extra JSON if needed (simplified for now)
	printerType := PrinterTypeMoonraker
	if printer.PrinterType != "" {
		printerType = PrinterType(printer.PrinterType)
	}

	switch printerType {
	case PrinterTypeBambuLab:
		config.PrinterType = PrinterTypeBambuLab
		provider = NewBambuProvider(config)
	case PrinterTypeMoonraker, PrinterTypeGeneric:
		fallthrough
	default:
		config.PrinterType = PrinterTypeMoonraker
		provider = NewMoonrakerProvider(config)
	}

	// Connect to printer
	ctx := context.Background()
	if err := provider.Connect(ctx); err != nil {
		log.Printf("Failed to connect to printer %s: %v", printer.Name, err)
		return nil
	}

	p.providers[printer.ID] = provider
	return provider
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
