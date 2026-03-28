// Package scanner provides network discovery for 3D printers
package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// PrinterType represents the type of discovered printer
type PrinterType string

const (
	PrinterTypeMoonraker PrinterType = "MOONRAKER"
	PrinterTypeBambuLab  PrinterType = "BAMBULAB"
	PrinterTypeUnknown   PrinterType = "UNKNOWN"
)

// DiscoveredPrinter represents a found printer on the network
type DiscoveredPrinter struct {
	IPAddress   string      `json:"ip_address"`
	Hostname    string      `json:"hostname"`
	PrinterType PrinterType `json:"printer_type"`
	Port        int         `json:"port"`
	Name        string      `json:"name"`
	Model       string      `json:"model"`
	Online      bool        `json:"online"`
}

// NetworkScanner discovers printers on the local network
type NetworkScanner struct {
	timeout time.Duration
}

// NewNetworkScanner creates a new network scanner
func NewNetworkScanner() *NetworkScanner {
	return &NetworkScanner{
		timeout: 2 * time.Second,
	}
}

// Scan discovers printers on the local network
func (s *NetworkScanner) Scan(ctx context.Context, subnet string) ([]DiscoveredPrinter, error) {
	log.Printf("[SCANNER] Starting network scan on subnet: %s", subnet)

	var wg sync.WaitGroup
	var mu sync.Mutex
	printers := []DiscoveredPrinter{}

	// Scan common printer ports
	ports := []int{7125, 8890, 80, 443} // Moonraker, Bambu, HTTP, HTTPS

	// Generate IPs from subnet
	ips := generateIPsFromSubnet(subnet)

	for _, ip := range ips {
		for _, port := range ports {
			wg.Add(1)
			go func(ip string, port int) {
				defer wg.Done()

				// Use net.JoinHostPort for proper IPv4/IPv6 handling
				address := net.JoinHostPort(ip, fmt.Sprintf("%d", port))
				conn, err := net.DialTimeout("tcp", address, s.timeout)
				if err != nil {
					return
				}
				conn.Close()

				// Port is open, identify the device
				printerType := identifyPrinterType(ctx, ip, port)
				if printerType != PrinterTypeUnknown {
					mu.Lock()
					printer := DiscoveredPrinter{
						IPAddress:   ip,
						Port:        port,
						PrinterType: printerType,
						Online:      true,
					}
					// Try to get more info
					printer.Name, printer.Model, printer.Hostname = getPrinterInfo(ctx, ip, port, printerType)
					printers = append(printers, printer)
					log.Printf("[SCANNER] Found %s printer at %s:%d (%s)", printerType, ip, port, printer.Name)
					mu.Unlock()
				}
			}(ip, port)
		}
	}

	// Wait for all scans to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		log.Printf("[SCANNER] Scan cancelled")
		return printers, nil
	case <-done:
		log.Printf("[SCANNER] Scan complete, found %d printers", len(printers))
	}

	// Sort by IP address
	sort.Slice(printers, func(i, j int) bool {
		return printers[i].IPAddress < printers[j].IPAddress
	})

	return printers, nil
}

// identifyPrinterType determines the type of printer by probing
func identifyPrinterType(ctx context.Context, ip string, port int) PrinterType {
	switch port {
	case 7125:
		return PrinterTypeMoonraker
	case 8890:
		return PrinterTypeBambuLab
	}

	// For HTTP ports, try to identify
	client := &http.Client{Timeout: 2 * time.Second}

	// Try Moonraker endpoint
	url := fmt.Sprintf("http://%s:%d/server/info", ip, port)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err == nil {
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return PrinterTypeMoonraker
		}
	}

	// Try Bambu endpoint
	url = fmt.Sprintf("https://%s:%d/api/v1/task", ip, port)
	req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
	if err == nil {
		req.Header.Set("access_code", "000000") // Dummy code for probe
		resp, err := client.Do(req)
		if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusForbidden) {
			resp.Body.Close()
			return PrinterTypeBambuLab
		}
	}

	return PrinterTypeUnknown
}

// getPrinterInfo fetches additional information about the printer
func getPrinterInfo(ctx context.Context, ip string, port int, printerType PrinterType) (name, model, hostname string) {
	client := &http.Client{Timeout: 2 * time.Second}

	switch printerType {
	case PrinterTypeMoonraker:
		url := fmt.Sprintf("http://%s:%d/server/info", ip, port)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				var result struct {
					Result struct {
						HostName string `json:"hostname"`
						Machine  string `json:"machine"`
					} `json:"result"`
				}
				json.NewDecoder(resp.Body).Decode(&result)
				resp.Body.Close()
				name = result.Result.Machine
				hostname = result.Result.HostName
			}
		}

	case PrinterTypeBambuLab:
		url := fmt.Sprintf("https://%s:%d/api/v1/task", ip, port)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err == nil {
			req.Header.Set("access_code", "000000")
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				var result struct {
					Print struct {
						File string `json:"gcode_file"`
					} `json:"print"`
				}
				json.NewDecoder(resp.Body).Decode(&result)
				resp.Body.Close()
				name = "Bambu Lab Printer"
				model = "Unknown"
			}
		}
	}

	return name, model, hostname
}

// generateIPsFromSubnet generates a list of IPs from a subnet CIDR
func generateIPsFromSubnet(subnet string) []string {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return []string{}
	}

	var ips []string
	ip := make(net.IP, len(ipNet.IP))
	copy(ip, ipNet.IP)

	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	return ips
}

// parseCIDR generates IPs from a CIDR notation (deprecated, use generateIPsFromSubnet)
func parseCIDR(cidr string) []string {
	return generateIPsFromSubnet(cidr)
}

// inc increments an IP address
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// ScanCommonSubnets scans common local subnets
func (s *NetworkScanner) ScanCommonSubnets(ctx context.Context) ([]DiscoveredPrinter, error) {
	commonSubnets := []string{
		"192.168.1.0/24",
		"192.168.0.0/24",
		"192.168.10.0/24",
		"192.168.100.0/24",
		"10.0.0.0/24",
	}

	var allPrinters []DiscoveredPrinter
	seen := make(map[string]bool)

	for _, subnet := range commonSubnets {
		printers, err := s.Scan(ctx, subnet)
		if err != nil {
			log.Printf("[SCANNER] Failed to scan %s: %v", subnet, err)
			continue
		}

		for _, p := range printers {
			if !seen[p.IPAddress] {
				seen[p.IPAddress] = true
				allPrinters = append(allPrinters, p)
			}
		}
	}

	return allPrinters, nil
}

// QuickScan performs a quick scan on the current subnet
func (s *NetworkScanner) QuickScan(ctx context.Context) ([]DiscoveredPrinter, error) {
	// Get local IP to determine subnet
	localIP, err := getLocalIP()
	if err != nil {
		return s.ScanCommonSubnets(ctx)
	}

	// Derive subnet from local IP
	subnet := deriveSubnet(localIP)
	return s.Scan(ctx, subnet)
}

// getLocalIP returns the local IP address
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no local IP found")
}

// deriveSubnet derives a /24 subnet from an IP
func deriveSubnet(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return "192.168.1.0/24"
	}
	return fmt.Sprintf("%s.%s.%s.0/24", parts[0], parts[1], parts[2])
}
