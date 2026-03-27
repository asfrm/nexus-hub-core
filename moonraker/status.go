package moonraker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PrinterStatus represents the current state of a 3D printer
type PrinterStatus struct {
	State        string  `json:"state"`
	Filename     string  `json:"filename"`
	Progress     float32 `json:"progress"`
	PrintTime    int64   `json:"print_time"`
	TimeLeft     int64   `json:"time_left"`
	Layer        int     `json:"layer"`
	TotalLayers  int     `json:"total_layers"`
	TempNozzle   float32 `json:"temp_nozzle"`
	TempBed      float32 `json:"temp_bed"`
	TargetNozzle float32 `json:"target_nozzle"`
	TargetBed    float32 `json:"target_bed"`
}

// PrinterInfo contains printer identification information
type PrinterInfo struct {
	HostName    string `json:"hostname"`
	Machine     string `json:"machine"`
	Version     string `json:"version"`
	KlipperVersion string `json:"klipper_version"`
}

// GetStatus fetches the current printer status from Moonraker
func GetStatus(printerIP string) (*PrinterStatus, error) {
	url := fmt.Sprintf("http://%s/printer/objects/query?print_stats&display_status&heater_bed&toolhead", printerIP)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to printer: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			PrintStats struct {
				State      string  `json:"state"`
				Filename   string  `json:"filename"`
				Progress   float32 `json:"progress"`
				PrintTime  float64 `json:"print_time"`
				TotalTime  float64 `json:"total_time"`
				Estimated  float64 `json:"filament_used"`
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
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	status := &PrinterStatus{
		State:     result.Result.PrintStats.State,
		Filename:  result.Result.PrintStats.Filename,
		Progress:  result.Result.DisplayStatus.Progress * 100,
		PrintTime: int64(result.Result.PrintStats.PrintTime),
		TempBed:   result.Result.HeaterBed.Temperature,
		TargetBed: result.Result.HeaterBed.Target,
	}

	// Calculate time left
	if result.Result.PrintStats.Progress > 0 && result.Result.PrintStats.TotalTime > 0 {
		elapsed := result.Result.PrintStats.PrintTime
		progress := result.Result.PrintStats.Progress
		if progress > 0 {
			total := elapsed / float64(progress)
			status.TimeLeft = int64(total - elapsed)
		}
	}

	return status, nil
}

// GetPrinterInfo fetches printer identification information
func GetPrinterInfo(printerIP string) (*PrinterInfo, error) {
	url := fmt.Sprintf("http://%s/server/info", printerIP)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
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
			HostName string `json:"hostname"`
			Machine  string `json:"machine"`
			Version  string `json:"version"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &PrinterInfo{
		HostName: result.Result.HostName,
		Machine:  result.Result.Machine,
		Version:  result.Result.Version,
	}, nil
}

// GetGCodeFiles lists available G-Code files on the printer
func GetGCodeFiles(printerIP string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s/server/files/list", printerIP)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
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
			Files []map[string]interface{} `json:"files"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Result.Files, nil
}
