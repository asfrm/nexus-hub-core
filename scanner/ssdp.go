// Package scanner provides SSDP discovery for Bambu Lab printers
package scanner

import (
	"bufio"
	"context"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// SSDP configuration for Bambu Lab printers
const (
	SSDPAddress = "239.255.255.250:1900"
	SSDPST      = "urn:bambulab-com:device:3dprinter:1"
	SSDPTimeout = 3 * time.Second
)

// DiscoveredPrinter represents a Bambu printer found via SSDP
type DiscoveredPrinter struct {
	IPAddress string `json:"ip_address"`
	Serial    string `json:"serial"`
	Model     string `json:"model"`
	Name      string `json:"name"`
	Online    bool   `json:"online"`
}

// Discover Bambu printers using SSDP
func Discover(ctx context.Context) ([]DiscoveredPrinter, error) {
	log.Printf("[SSDP] Starting Bambu printer discovery...")

	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(SSDPTimeout))

	// M-SEARCH request
	request := "M-SEARCH * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1900\r\n" +
		"MAN: \"ssdp:discover\"\r\n" +
		"MX: 2\r\n" +
		"ST: urn:bambulab-com:device:3dprinter:1\r\n" +
		"\r\n"

	_, err = conn.WriteTo([]byte(request), &net.UDPAddr{
		IP:   net.ParseIP("239.255.255.250"),
		Port: 1900,
	})
	if err != nil {
		return nil, err
	}

	var mu sync.Mutex
	printers := make(map[string]DiscoveredPrinter)
	buffer := make([]byte, 4096)
	done := make(chan bool)

	go func() {
		for {
			n, addr, err := conn.ReadFrom(buffer)
			if err != nil {
				done <- true
				return
			}

			printer := parseSSDPResponse(string(buffer[:n]), addr.String())
			if printer.Serial != "" {
				mu.Lock()
				printers[printer.Serial] = printer
				mu.Unlock()
				log.Printf("[SSDP] Found: %s at %s (serial: %s)",
					printer.Name, printer.IPAddress, printer.Serial)
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(SSDPTimeout + 1*time.Second):
	case <-ctx.Done():
	}

	result := make([]DiscoveredPrinter, 0, len(printers))
	for _, p := range printers {
		result = append(result, p)
	}

	log.Printf("[SSDP] Discovery complete: found %d printers", len(result))
	return result, nil
}

func parseSSDPResponse(response, addr string) DiscoveredPrinter {
	printer := DiscoveredPrinter{Online: true}
	
	if host, _, err := net.SplitHostPort(addr); err == nil {
		printer.IPAddress = host
	}

	scanner := bufio.NewScanner(strings.NewReader(response))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if strings.HasPrefix(line, "LOCATION:") {
			location := strings.TrimSpace(strings.TrimPrefix(line, "LOCATION:"))
			if ip := extractIP(location); ip != "" {
				printer.IPAddress = ip
			}
		}
		
		if strings.HasPrefix(line, "USN:") {
			usn := strings.TrimSpace(strings.TrimPrefix(line, "USN:"))
			printer.Serial = extractSerial(usn)
		}
		
		if strings.HasPrefix(line, "SERVER:") {
			server := strings.TrimSpace(strings.TrimPrefix(line, "SERVER:"))
			printer.Model = extractModel(server)
			printer.Name = printer.Model
		}
	}

	return printer
}

func extractIP(location string) string {
	if strings.Contains(location, "http://") {
		parts := strings.Split(strings.TrimPrefix(location, "http://"), "/")
		if len(parts) > 0 {
			return strings.Split(parts[0], ":")[0]
		}
	}
	return ""
}

func extractSerial(usn string) string {
	if strings.Contains(usn, "uuid:bblp-3dprinter-") {
		parts := strings.Split(usn, "::")
		if len(parts) > 0 {
			return strings.TrimPrefix(parts[0], "uuid:bblp-3dprinter-")
		}
	}
	return ""
}

func extractModel(server string) string {
	parts := strings.SplitN(server, " ", 3)
	if len(parts) >= 3 {
		return parts[2]
	}
	return "Bambu Lab Printer"
}
