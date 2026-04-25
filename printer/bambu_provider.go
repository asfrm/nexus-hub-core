// Package printer provides Bambu Lab printer integration via bambusdk-go
package printer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/asfrm/bambusdk-go/filament"
	"github.com/asfrm/bambusdk-go/printer"
	"github.com/asfrm/bambusdk-go/states"
)

// BambuProvider implements PrinterProvider for Bambu Lab printers using bambusdk-go
type BambuProvider struct {
	config     PrinterConfig
	printer    *printer.Printer
	mu         sync.Mutex
	caps       PrinterCapabilities
	lastStatus *PrinterStatus
}

// NewBambuProvider creates a new Bambu Lab printer provider
func NewBambuProvider(config PrinterConfig) *BambuProvider {
	bp := &BambuProvider{
		config: config,
		caps: PrinterCapabilities{
			HasCamera:       true,
			HasLEDControl:   true,
			HasFanControl:   true,
			HasSpeedControl: true,
			HasLaser:        false,
			MaxExtruders:    1,
		},
	}

	// Extract connection details from Extra map
	ipAddress := config.Extra["local_ip"]
	if ipAddress == "" {
		ipAddress = config.Address
	}
	accessCode := config.Extra["access_code"]
	serial := config.Extra["serial"]

	bp.printer = printer.NewPrinter(ipAddress, accessCode, serial)

	return bp
}

// Connect establishes connection to the Bambu printer
func (b *BambuProvider) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.printer.MQTTClientConnected() {
		return nil
	}

	log.Printf("[BAMBU] Connecting to %s (serial: %s)", b.config.Extra["local_ip"], b.config.Extra["serial"])

	if err := b.printer.Connect(); err != nil {
		return fmt.Errorf("failed to connect to Bambu printer: %w", err)
	}

	log.Printf("[BAMBU] Connected successfully")
	return nil
}

// Disconnect closes the connection
func (b *BambuProvider) Disconnect() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.printer.Disconnect()
	log.Printf("[BAMBU] Disconnected")
	return nil
}

// IsConnected returns connection status
func (b *BambuProvider) IsConnected() bool {
	return b.printer.MQTTClientConnected()
}

// GetStatus returns current printer status
func (b *BambuProvider) GetStatus(ctx context.Context) (*PrinterStatus, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	status := &PrinterStatus{
		PrinterID:    b.config.PrinterID,
		Temperatures: make(map[string]float32),
		Targets:      make(map[string]float32),
		Capabilities: b.caps,
		VendorData:   make(map[string]interface{}),
	}

	// Get G-code state
	gcodeState := b.printer.GetState()
	status.State = b.mapGcodeState(gcodeState)

	// Get print status (hardware state)
	printStatus := b.printer.GetCurrentState()
	status.VendorData["print_status"] = printStatus.String()
	status.VendorData["print_status_code"] = int(printStatus)

	// Progress and time
	status.Progress = float32(b.printer.GetPercentage())
	status.TimeLeft = int64(b.printer.GetTime())

	// Layers
	status.Layer = b.printer.CurrentLayerNum()
	status.TotalLayers = b.printer.TotalLayerNum()

	// File
	status.Filename = b.printer.GcodeFile()

	// Temperatures
	status.Temperatures["nozzle"] = float32(b.printer.GetNozzleTemperature())
	status.Temperatures["bed"] = float32(b.printer.GetBedTemperature())

	chamberTemp := b.printer.GetChamberTemperature()
	if chamberTemp > 0 {
		status.Temperatures["chamber"] = float32(chamberTemp)
	}

	// Fan speed (0-255 to percentage)
	fanGear := b.printer.GetFanGear()
	partFanSpeed := fanGear % 256
	status.FanSpeed = float32(partFanSpeed) / 255.0 * 100.0

	// Speed level
	status.SpeedLevel = b.printer.GetPrintSpeed()

	// Light state
	status.VendorData["light_state"] = b.printer.GetLightState()
	status.VendorData["wifi_signal"] = b.printer.WifiSignal()

	// --- FIXED: PERFECTLY INDEXED AMS SLOTS USING SLICE INDICES ---
	rawAms := b.printer.AMSHub()

	// 1. Determine how many slots we need based on connected AMS units (minimum 4)
	maxSlots := 4
	if rawAms != nil && len(rawAms.AMSHub) > 0 {
		maxSlots = len(rawAms.AMSHub) * 4
	}

	// 2. Pre-fill the array with empty slots so the indexes never shift
	spools := make([]map[string]interface{}, maxSlots)
	for i := 0; i < maxSlots; i++ {
		spools[i] = map[string]interface{}{
			"color_hex": "#555555",
			"material":  "Empty",
			"empty":     true,
		}
	}

	// 3. Populate only the slots that actually have filament
	if rawAms != nil {
		for amsIdx, amsUnit := range rawAms.AMSHub {
			for trayIdx, tray := range amsUnit.FilamentTrays {

				// Calculate absolute slot ID (0-3 for AMS 1, 4-7 for AMS 2, etc.)
				slotID := (int(amsIdx) * 4) + int(trayIdx)

				if slotID >= 0 && slotID < maxSlots {
					color := tray.TrayColor
					if len(color) >= 6 {
						color = "#" + color[:6]
					} else {
						color = "#555555"
					}

					materialName := tray.TrayInfoIdx
					if tray.TrayInfoIdx != "" {
						if fil, err := filament.FilamentByIndex(tray.TrayInfoIdx); err == nil {
							materialName = fil.Name()
						}
					}

					// Insert the filament into its TRUE physical index
					spools[slotID] = map[string]interface{}{
						"color_hex": color,
						"material":  materialName,
						"empty":     tray.TrayInfoIdx == "",
					}
				}
			}
		}
	}

	// Wrap it in a "spools" object so the frontend JS condition passes
	status.VendorData["ams"] = map[string]interface{}{
		"spools": spools,
	}

	b.lastStatus = status
	return status, nil
}

// mapGcodeState maps SDK gcode state to our state strings
func (b *BambuProvider) mapGcodeState(state states.GcodeState) string {
	switch state {
	case states.GcodeStateRunning, states.GcodeStatePrepare:
		return "PRINTING"
	case states.GcodeStatePause:
		return "PAUSED"
	case states.GcodeStateFinish:
		return "COMPLETE"
	case states.GcodeStateFailed:
		return "ERROR"
	default:
		return "IDLE"
	}
}

// SendCommand sends a control command to the printer
func (b *BambuProvider) SendCommand(ctx context.Context, cmd PrinterCommand) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.printer.MQTTClientConnected() {
		return fmt.Errorf("not connected to printer")
	}

	// Hardware safety check - don't send G-code while printing
	if cmd.Type == "gcode" {
		if b.printer.IsBusy() {
			return fmt.Errorf("hardware safety lock: cannot send G-code while printer is busy (%s)",
				b.printer.GetActivityDescription())
		}
	}

	var err error
	switch cmd.Type {
	case "pause":
		err = b.printer.PausePrint()
	case "resume":
		err = b.printer.ResumePrint()
	case "stop":
		err = b.printer.StopPrint()
	case "g28", "home":
		err = b.printer.HomePrinter()
	case "set_fan":
		speed := 0
		if s, ok := cmd.Params["speed"].(float64); ok {
			speed = int(s * 255 / 100)
		}
		_, err = b.printer.SetPartFanSpeed(speed)
	case "set_speed":
		level := 0
		if l, ok := cmd.Params["level"].(int); ok {
			level = l
		}
		err = b.printer.SetPrintSpeed(level)
	case "light_on":
		err = b.printer.TurnLightOn()
	case "light_off":
		err = b.printer.TurnLightOff()
	case "gcode":
		if gcode, ok := cmd.Params["gcode"].(string); ok {
			_, err = b.printer.Gcode(gcode, false)
		}
	default:
		return fmt.Errorf("unsupported command: %s", cmd.Type)
	}

	return err
}

// UploadAndPrint uploads a file and starts printing
func (b *BambuProvider) UploadAndPrint(ctx context.Context, job PrintJob) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.printer.MQTTClientConnected() {
		return fmt.Errorf("not connected to printer")
	}

	// Prepare temp path for FTP upload
	tempPath := filepath.Join("uploads", fmt.Sprintf("temp_%d_%s", time.Now().UnixNano(), job.Filename))
	if err := os.MkdirAll("uploads", 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	if err := os.WriteFile(tempPath, job.FileData, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	defer os.Remove(tempPath)

	file, err := os.Open(tempPath)
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer file.Close()

	// --- ROBUST AMS MAPPING FIX ---
	useAMS := false
	// Bambu printers use a mapping array. Default to -1 (unused).
	finalMapping := make([]int, 16)
	for i := range finalMapping {
		finalMapping[i] = -1
	}

	if job.Metadata != nil {
		if val, ok := job.Metadata["use_ams"].(bool); ok {
			useAMS = val
		}

		var selectedPhysicalSlot int = -1
		// Extract the slot index you clicked in the UI
		if val, ok := job.Metadata["ams_mapping"].([]int); ok && len(val) > 0 {
			selectedPhysicalSlot = val[0]
		} else if val, ok := job.Metadata["ams_mapping"].([]interface{}); ok && len(val) > 0 {
			if floatVal, ok := val[0].(float64); ok {
				selectedPhysicalSlot = int(floatVal)
			}
		}

		// If a slot was selected (0, 1, 2, or 3)
		if selectedPhysicalSlot != -1 {
			// FORCE all virtual trays to point to your selected physical slot.
			// This bypasses any internal indexing errors in the .3mf file.
			for i := 0; i < 16; i++ {
				finalMapping[i] = selectedPhysicalSlot
			}
		}
	}

	log.Printf("[BAMBU] Triggering print: %s | AMS: %v | Mapping: %v", job.Filename, useAMS, finalMapping)

	// Submit print job via FTP + MQTT
	_, err = b.printer.SubmitPrintJob(
		file,
		job.Filename,
		1,            // Plate 1
		useAMS,       // Use AMS boolean
		finalMapping, // The 16-slot mapping array
		false,        // flow calibration (can set to true if desired)
		"",           // bed type (auto-detect)
	)

	return err
}

// GetFileInfo retrieves information about a file
func (b *BambuProvider) GetFileInfo(ctx context.Context, filepath string) (map[string]interface{}, error) {
	// Bambu printers store files internally, this is a stub
	return map[string]interface{}{
		"name": filepath,
		"type": "local",
	}, nil
}

// ListFiles returns available files on the printer
func (b *BambuProvider) ListFiles(ctx context.Context) ([]string, error) {
	// Return empty list - Bambu doesn't expose file listing easily
	return []string{}, nil
}

// GetCapabilities returns printer capabilities
func (b *BambuProvider) GetCapabilities() PrinterCapabilities {
	return b.caps
}

// GetPrinter returns the underlying SDK printer instance (for advanced usage)
func (b *BambuProvider) GetPrinter() *printer.Printer {
	return b.printer
}

// GetMQTTDump returns raw MQTT data dump (for debugging)
func (b *BambuProvider) GetMQTTDump() map[string]interface{} {
	dump := b.printer.MQTTDump()
	result := make(map[string]interface{})
	for k, v := range dump {
		result[k] = v
	}
	return result
}

// GetMQTTDumpJSON returns raw MQTT data dump as JSON string (for debugging)
func (b *BambuProvider) GetMQTTDumpJSON() (string, error) {
	dump := b.printer.MQTTDump()
	jsonBytes, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
