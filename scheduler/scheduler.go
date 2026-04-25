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

// Scheduler handles automatic job assignment to printers with wear-leveling
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

	// Get wear stats for all printers
	wearStats, err := s.database.GetPrinterWearStats()
	if err != nil {
		log.Printf("Failed to get wear stats: %v", err)
		return
	}

	// Build wear score map
	wearScoreMap := make(map[int64]float64)
	for _, stat := range wearStats {
		wearScoreMap[stat.PrinterID] = stat.WearScore
	}

	// Get available clusters and printers
	clusters, err := s.database.GetClustersWithPrinters()
	if err != nil {
		log.Printf("Failed to get clusters: %v", err)
		return
	}

	// Try to assign each pending job
	for _, job := range pendingJobs {
		assigned := s.tryAssignJobWithWearLeveling(job, clusters, wearScoreMap)
		if assigned {
			break // Assign one job per cycle to avoid overwhelming
		}
	}
}

// tryAssignJobWithWearLeveling assigns a job using wear-balanced selection
func (s *Scheduler) tryAssignJobWithWearLeveling(job db.PrintJob, clusters []db.Cluster, wearScoreMap map[int64]float64) bool {
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

	// Find idle printers and select the one with lowest wear score
	var bestPrinter *db.Printer
	bestWearScore := float64(-1)

	for i, printer := range matchingCluster.Printers {
		if printer.Status != "IDLE" {
			continue
		}

		wearScore := wearScoreMap[printer.ID]

		// Select printer with lowest wear score (least used)
		if bestPrinter == nil || wearScore < bestWearScore {
			bestPrinter = &matchingCluster.Printers[i]
			bestWearScore = wearScore
		}
	}

	if bestPrinter == nil {
		log.Printf("No idle printer in cluster: %s", matchingCluster.Name)
		return false
	}

	// Assign the job
	log.Printf("Auto-assigning job '%s' to printer '%s' (wear score: %.2f) in cluster '%s'",
		job.Filename, bestPrinter.Name, bestWearScore, matchingCluster.Name)

	err := s.database.AssignJob(job.ID, matchingCluster.ID, bestPrinter.ID)
	if err != nil {
		log.Printf("Failed to assign job: %v", err)
		return false
	}

	// Log the assignment
	s.hub.Send(api.NexusPacket{
		EventType: "SYSTEM_LOG",
		Log: &api.LogPayload{
			Module: "SCHEDULER",
			Level:  "INFO",
			Message: fmt.Sprintf("Wear-balanced assignment: '%s' → %s (score: %.1f) in %s",
				job.Filename, bestPrinter.Name, bestWearScore, matchingCluster.Name),
			Timestamp: time.Now(),
		},
	})

	return true
}
