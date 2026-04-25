package web

import (
	"encoding/json"
	"log"
	"net/http"
	"nexus-hub-core/api"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleWS(hub *api.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WS Upgrade error: %v", err)
			return
		}
		defer conn.Close()

		// Subscribe to hub broadcasts
		subChan := hub.Subscribe()

		// Handle ping/pong for connection health
		conn.SetPingHandler(func(appData string) error {
			return conn.WriteMessage(websocket.PongMessage, []byte(appData))
		})

		// Send ping every 30 seconds to detect dead connections
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					return
				}
			}
		}()

		for {
			select {
			case packet, ok := <-subChan:
				if !ok {
					return // Channel closed
				}
				data, err := json.Marshal(packet)
				if err != nil {
					continue
				}
				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					return
				}
			}
		}
	}
}
