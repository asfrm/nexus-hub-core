package web

import (
	"encoding/json"
	"log"
	"net/http"
	"nexus-hub-core/api"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Аналог CORS Allow Any
}

func HandleWS(tx chan api.NexusPacket) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WS Upgrade error: %v", err)
			return
		}
		defer conn.Close()

		// В Rust ты делал subscribe(). В Go мы просто читаем из канала.
		// Важно: если у тебя много вкладок браузера, тебе понадобится
		// простой менеджер подписчиков (Hub), но для начала хватит и одного.
		for packet := range tx {
			data, _ := json.Marshal(packet)
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				break
			}
		}
	}
}
