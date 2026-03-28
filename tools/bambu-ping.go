package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: go run bambu-ping.go <IP> <ACCESS_CODE> <SERIAL>")
		return
	}

	ip := os.Args[1]
	pass := os.Args[2]
	serial := os.Args[3]

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcps://%s:8883", ip))
	opts.SetClientID("nexus-debug-homing")
	opts.SetUsername("bblp")
	opts.SetPassword(pass)
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("❌ ОШИБКА ПОДКЛЮЧЕНИЯ: %v\n", token.Error())
		return
	}
	fmt.Printf("✅ Подключились к %s!\n", ip)

	requestTopic := fmt.Sprintf("device/%s/request", serial)

	// Формируем JSON-пакет для отправки G-Code G28 (Homing)
	payload := `{"print": {"sequence_id": "1", "command": "gcode_line", "param": "G28 \n"}}`

	fmt.Printf("🚀 ОТПРАВЛЯЕМ КОМАНДУ ПАРКОВКИ (G28)...\n")

	// Публикуем с QOS 1, чтобы гарантировать доставку
	token := client.Publish(requestTopic, 1, false, payload)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("❌ ОШИБКА ОТПРАВКИ: %v\n", token.Error())
	} else {
		fmt.Println("💥 КОМАНДА УШЛА! СМОТРИ НА ПРИНТЕР!")
	}

	// Ждем немного, чтобы пакет точно долетел перед закрытием соединения
	time.Sleep(3 * time.Second)
	client.Disconnect(250)
	fmt.Println("🏁 Щуп завершил работу.")
}
