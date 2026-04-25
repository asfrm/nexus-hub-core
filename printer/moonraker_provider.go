// Package printer provides integration with Klipper/Moonraker 3D printers
package printer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MoonrakerProvider implements printer.Provider for Moonraker/Klipper printers
type MoonrakerProvider struct {
	config     PrinterConfig
	httpClient *http.Client
	baseURL    string
	mu         sync.Mutex
	connected  bool
	lastStatus *PrinterStatus
}

// NewMoonrakerProvider creates a new Moonraker printer provider
func NewMoonrakerProvider(config PrinterConfig) *MoonrakerProvider {
	port := config.Port
	if port == 0 {
		port = 80
	}

	return &MoonrakerProvider{
		config:  config,
		baseURL: fmt.Sprintf("http://%s:%d", config.Address, port),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Connect establishes connection to Moonraker
func (m *MoonrakerProvider) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Test connection by getting status
	_, err := m.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	m.connected = true
	return nil
}

// Disconnect closes the connection
func (m *MoonrakerProvider) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

// IsConnected returns connection status
func (m *MoonrakerProvider) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// GetStatus fetches current printer status
func (m *MoonrakerProvider) GetStatus(ctx context.Context) (*PrinterStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	url := fmt.Sprintf("%s/printer/objects/query?print_stats&display_status&heater_bed&toolhead&fan", m.baseURL)

	resp, err := m.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			PrintStats struct {
				State     string  `json:"state"`
				Filename  string  `json:"filename"`
				Progress  float32 `json:"progress"`
				PrintTime float64 `json:"print_time"`
				TotalTime float64 `json:"total_time"`
			} `json:"print_stats"`
			DisplayStatus struct {
				Progress float32 `json:"progress"`
			} `json:"display_status"`
			HeaterBed struct {
				Temperature float32 `json:"temperature"`
				Target      float32 `json:"target"`
			} `json:"heater_bed"`
			Toolhead struct {
				EstimatedPrintTime float64 `json:"estimated_print_time"`
			} `json:"toolhead"`
			Fan struct {
				Speed float32 `json:"speed"`
			} `json:"fan"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	state := "idle"
	switch result.Result.PrintStats.State {
	case "printing":
		state = "printing"
	case "paused":
		state = "paused"
	case "complete":
		state = "complete"
	case "error":
		state = "error"
	case "standby":
		state = "idle"
	}

	// Calculate time left
	timeLeft := int64(0)
	if result.Result.PrintStats.Progress > 0 && result.Result.PrintStats.TotalTime > 0 {
		elapsed := result.Result.PrintStats.PrintTime
		progress := result.Result.PrintStats.Progress
		if progress > 0 {
			total := elapsed / float64(progress)
			timeLeft = int64(total - elapsed)
		}
	}

	status := &PrinterStatus{
		PrinterID: m.config.PrinterID,
		State:     state,
		Filename:  result.Result.PrintStats.Filename,
		Progress:  result.Result.DisplayStatus.Progress * 100,
		PrintTime: int64(result.Result.PrintStats.PrintTime),
		TimeLeft:  timeLeft,
		Temperatures: map[string]float32{
			"nozzle": 0,
			"bed":    result.Result.HeaterBed.Temperature,
		},
		Targets: map[string]float32{
			"nozzle": 0,
			"bed":    result.Result.HeaterBed.Target,
		},
		FanSpeed: result.Result.Fan.Speed * 100,
		Capabilities: PrinterCapabilities{
			HasCamera:       false,
			HasLEDControl:   false,
			HasFanControl:   true,
			HasSpeedControl: true,
			HasLaser:        false,
			MaxExtruders:    1,
		},
		VendorData: map[string]interface{}{
			"estimated_time": result.Result.Toolhead.EstimatedPrintTime,
		},
	}

	m.lastStatus = status
	return status, nil
}

// SendCommand sends a control command
func (m *MoonrakerProvider) SendCommand(ctx context.Context, cmd PrinterCommand) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var endpoint string
	var payload interface{}

	switch cmd.Type {
	case "pause":
		endpoint = "/printer/print/pause"
	case "resume":
		endpoint = "/printer/print/resume"
	case "stop":
		endpoint = "/printer/print/cancel"
	case "estop":
		endpoint = "/printer/emergency_stop"
	case "set_fan":
		endpoint = "/printer/gcode/script"
		speed := 0
		if s, ok := cmd.Params["speed"].(float64); ok {
			speed = int(s * 255 / 100)
		}
		payload = map[string]string{"script": fmt.Sprintf("M106 S%d", speed)}
	case "set_temp":
		endpoint = "/printer/gcode/script"
		target := 0
		if t, ok := cmd.Params["target"].(float64); ok {
			target = int(t)
		}
		payload = map[string]string{"script": fmt.Sprintf("M109 S%d", target)}
	default:
		return fmt.Errorf("unknown command: %s", cmd.Type)
	}

	url := m.baseURL + endpoint
	var req *http.Request
	var err error

	if payload != nil {
		jsonData, _ := json.Marshal(payload)
		req, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	} else {
		req, err = http.NewRequestWithContext(ctx, "POST", url, nil)
	}

	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("command failed: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// UploadAndPrint uploads a file and starts printing
func (m *MoonrakerProvider) UploadAndPrint(ctx context.Context, job PrintJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, err := os.Open(filepath.Join("gcode_vault", job.Filename))
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(job.Filename))
	if err != nil {
		return err
	}
	io.Copy(part, file)

	// Flag 'print=true' starts printing immediately
	_ = writer.WriteField("print", "true")
	writer.Close()

	url := fmt.Sprintf("%s/server/files/upload", m.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetFileInfo retrieves information about a file
func (m *MoonrakerProvider) GetFileInfo(ctx context.Context, filepath string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/server/files/metadata/%s", m.baseURL, strings.TrimPrefix(filepath, "/"))

	resp, err := m.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result map[string]interface{} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Result, nil
}

// ListFiles returns available files
func (m *MoonrakerProvider) ListFiles(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/server/files/list?root=gcodes", m.baseURL)

	resp, err := m.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			Files []struct {
				Path string `json:"path"`
			} `json:"files"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	files := make([]string, len(result.Result.Files))
	for i, f := range result.Result.Files {
		files[i] = f.Path
	}

	return files, nil
}

// GetCapabilities returns printer capabilities
func (m *MoonrakerProvider) GetCapabilities() PrinterCapabilities {
	return PrinterCapabilities{
		HasCamera:       false,
		HasLEDControl:   false,
		HasFanControl:   true,
		HasSpeedControl: true,
		HasLaser:        false,
		MaxExtruders:    1,
	}
}
