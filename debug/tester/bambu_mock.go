package main

import (
	"fmt"
	"log"
	"net"
)

type BambuMock struct {
	IP   string
	Port int
}

func StartBambuMock(ip string, port int) *BambuMock {
	b := &BambuMock{IP: ip, Port: port}
	
	// Mock MQTT (port 1883)
	go func() {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", 1883))
		if err != nil {
			log.Printf("[Bambu Mock MQTT] Error: %v", err)
			return
		}
		log.Printf("[Bambu Mock MQTT] Listening on :1883")
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			// Just accept and keep open for now
			log.Printf("[Bambu Mock MQTT] Accepted connection from %s", conn.RemoteAddr())
			go handleMqttConn(conn)
		}
	}()

	// Mock FTP (port 21)
	go func() {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", 21))
		if err != nil {
			log.Printf("[Bambu Mock FTP] Error: %v", err)
			return
		}
		log.Printf("[Bambu Mock FTP] Listening on :21")
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			log.Printf("[Bambu Mock FTP] Accepted connection from %s", conn.RemoteAddr())
			go handleFtpConn(conn)
		}
	}()

	return b
}

func handleMqttConn(conn net.Conn) {
	defer conn.Close()
	// Minimal MQTT handshake could be implemented here if needed
	// For now, we just stay connected
	buf := make([]byte, 1024)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			return
		}
	}
}

func handleFtpConn(conn net.Conn) {
	defer conn.Close()
	conn.Write([]byte("220 Bambu FTP Mock Ready\r\n"))
	buf := make([]byte, 1024)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			return
		}
		// Respond to common FTP commands to avoid SDK hang
		conn.Write([]byte("230 Login successful\r\n"))
	}
}
