package scheduler

import (
	"context"
	"fmt"
	"log"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"sync/atomic"
	"time"
)

// Scheduler handles automatic job assignment to printers
type Scheduler struct {
	database *db.Database
	hub      *api.Hub
	autoMode *atomic.Bool
	interval time.Duration
}

// NewScheduler creates a new job scheduler
func NewScheduler(database *db.Database, hub *api.Hub, autoMode *atomic.Bool) *Scheduler {
	return &Scheduler{
		database: database,
		hub:      hub,
		autoMode: autoMode,
		interval: 10 * time.Second,
	}
}

// Start begins the scheduling loop
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Job scheduler shutting down...")
			return
		case <-ticker.C:
			if s.autoMode.Load() {
				s.runScheduler()
			}
		}
	}
}

func (s *Scheduler) runScheduler() {
	// Get pending jobs
	pendingJobs, err := s.database.GetPendingJobs()
	if err != nil {
		log.Printf("Failed to get pending jobs: %v", err)
		return
	}

	if len(pendingJobs) == 0 {
		return
	}

	// Get available clusters and printers
	clusters, err := s.database.GetClustersWithPrinters()
	if err != nil {
		log.Printf("Failed to get clusters: %v", err)
		return
	}

	// Try to assign each pending job
	for _, job := range pendingJobs {
		assigned := s.tryAssignJob(job, clusters)
		if assigned {
			break // Assign one job per cycle to avoid overwhelming
		}
	}
}

func (s *Scheduler) tryAssignJob(job db.PrintJob, clusters []db.Cluster) bool {
	// Find matching cluster for this printer model
	var matchingCluster *db.Cluster
	for i, cluster := range clusters {
		if !cluster.IsEnabled {
			continue
		}
		if cluster.TargetModel == job.PrinterModel {
			matchingCluster = &clusters[i]
			break
		}
	}

	if matchingCluster == nil {
		log.Printf("No cluster found for printer model: %s", job.PrinterModel)
		return false
	}

	// Find an idle printer in the cluster
	var idlePrinter *db.Printer
	for i, printer := range matchingCluster.Printers {
		if printer.Status == "IDLE" {
			idlePrinter = &matchingCluster.Printers[i]
			break
		}
	}

	if idlePrinter == nil {
		log.Printf("No idle printer in cluster: %s", matchingCluster.Name)
		return false
	}

	// Assign the job
	log.Printf("Auto-assigning job '%s' to printer '%s' in cluster '%s'",
		job.Filename, idlePrinter.Name, matchingCluster.Name)

	err := s.database.AssignJob(job.ID, matchingCluster.ID, idlePrinter.ID)
	if err != nil {
		log.Printf("Failed to assign job: %v", err)
		return false
	}

	// Log the assignment
	s.hub.Send(api.NexusPacket{
		EventType: "SYSTEM_LOG",
		Log: &api.LogPayload{
			Module:    "SCHEDULER",
			Level:     "INFO",
			Message: fmt.Sprintf("Auto-assigned '%s' to %s (%s)",
				job.Filename, idlePrinter.Name, matchingCluster.Name),
			Timestamp: time.Now(),
		},
	})

	return true
}
