package printer

import (
	"context"
	"fmt"
	"log"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"nexus-hub-core/moonraker"
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
}

// NewPoller creates a new printer status poller
func NewPoller(database *db.Database, hub *api.Hub) *Poller {
	return &Poller{
		database: database,
		hub:      hub,
		interval: 5 * time.Second,
		stopCh:   make(chan struct{}),
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
			p.pollAllPrinters()
		}
	}
}

func (p *Poller) pollAllPrinters() {
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
			go p.pollPrinter(printer.ID, printer.IPAddress, printer.Status)
		}
	}
}

func (p *Poller) pollPrinter(printerID int64, ipAddress, currentStatus string) {
	status, err := moonraker.GetStatus(ipAddress)
	if err != nil {
		// Printer unreachable - mark as error
		if currentStatus != "ERROR" {
			p.database.UpdatePrinterStatus(printerID, "ERROR")
			p.hub.Send(api.NexusPacket{
				EventType: "SYSTEM_LOG",
				Log: &api.LogPayload{
					Module:    "POLLER",
					Level:     "ERROR",
					Message:   fmt.Sprintf("Printer %s unreachable: %v", ipAddress, err),
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
	if newStatus != currentStatus {
		p.database.UpdatePrinterStatus(printerID, newStatus)

		// If print completed, update job status
		if newStatus == "IDLE" && currentStatus == "PRINTING" {
			p.markJobComplete(printerID)
		}

		p.hub.Send(api.NexusPacket{
			EventType: "SYSTEM_LOG",
			Log: &api.LogPayload{
				Module:    "POLLER",
				Level:     "INFO",
				Message:   fmt.Sprintf("Printer %s status changed: %s -> %s", ipAddress, currentStatus, newStatus),
				Timestamp: time.Now(),
			},
		})
	}

	// Broadcast printer status update
	p.hub.Send(api.NexusPacket{
		EventType: "PRINTER_STATUS",
		Payload: map[string]interface{}{
			"printer_id":  printerID,
			"status":      newStatus,
			"filename":    status.Filename,
			"progress":    status.Progress,
			"time_left":   status.TimeLeft,
			"temp_nozzle": status.TempNozzle,
			"temp_bed":    status.TempBed,
		},
	})
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
