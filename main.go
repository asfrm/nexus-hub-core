package main

import (
	"flag"
	"log"
	"nexus-hub-core/api"
	"nexus-hub-core/db"
	"nexus-hub-core/dispatcher"
	"nexus-hub-core/monitor"
	"nexus-hub-core/web"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 1. Флаги запуска
	isDebug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	// 2. Инициализация БД (SQLite)
	database, err := db.NewDatabase("hub.db")
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	// 3. Каналы (Channels) — замена mpsc и broadcast
	// internalChan — для сбора данных
	internalChan := make(chan api.NexusPacket, 100)

	// Для широковещания (broadcast) в Go обычно используют список подписчиков
	// или простую обертку, но для начала сделаем базовую логику
	broadcastChan := make(chan api.NexusPacket, 100)

	// 4. Воркер логирования и распределения (аналог твоего токио-спавна)
	go func() {
		for packet := range internalChan {
			// Сохраняем в БД
			_ = database.SaveEvent(packet)

			// Отправляем в бродкаст (всем, кто слушает)
			select {
			case broadcastChan <- packet:
			default:
				// Если канал полон, пропускаем, чтобы не блокировать систему
			}
		}
	}()

	// 5. Запуск модулей (Goroutines вместо tokio::spawn)
	if *isDebug {
		go debugLogger(broadcastChan)
	}

	go monitor.StartLocalCollection(internalChan)
	go web.StartWebEngine(database, broadcastChan, 8080)
	go dispatcher.StartDispatching(database, broadcastChan)

	// 6. Ожидание завершения (Ctrl+C)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down Nexus Hub...")
}

func debugLogger(ch chan api.NexusPacket) {
	for packet := range ch {
		log.Printf("[DEBUG] Received packet: %s", packet.EventType)
	}
}
