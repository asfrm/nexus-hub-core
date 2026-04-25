package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	// --- CONFIG ---
	ip := "192.168.1.136"
	serial := "0300DA5C0900350"
	accessCode := "779532da"
	// --------------

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tls://%s:8883", ip))
	opts.SetUsername("bblp")
	opts.SetPassword(accessCode)
	opts.SetClientID("nexus-debug-sniffer")
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

	opts.OnConnect = func(c mqtt.Client) {
		fmt.Println("✅ CONNECTED. Watching for OrcaSlicer traffic...")
		fmt.Println("👉 Now go to OrcaSlicer and hit 'Print' normally.")

		// Subscribe to EVERYTHING under the device topic
		topic := fmt.Sprintf("device/%s/#", serial)
		c.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
			fmt.Printf("\n--- MESSAGE ON %s ---\n%s\n", msg.Topic(), string(msg.Payload()))
		})
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("❌ Connection failed: %v\n", token.Error())
		return
	}

	// Wait for Ctrl+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
