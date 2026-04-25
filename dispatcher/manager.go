package dispatcher

import (
	"context"
	"log"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"sync/atomic"
	"time"
)

// AutoMode controls automatic job assignment (true = Auto, false = Manual)
var AutoMode atomic.Bool

func StartDispatching(ctx context.Context, database *db.Database, hub *api.Hub) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Dispatcher shutting down...")
			return
		case <-ticker.C:
			// Periodic maintenance tasks
			log.Printf("Dispatcher heartbeat - AutoMode: %v", AutoMode.Load())
		}
	}
}
