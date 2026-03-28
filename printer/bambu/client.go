// Package bambu provides Bambu Lab printer integration via native MQTT
// Based on official Bambu Lab protocol reverse-engineering
package bambu

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Bambu Lab MQTT brokers
const (
	BambuMQTTUS    = "tls://us.mqtt.bambulab.com:8883"
	BambuMQTTEU    = "tls://eu.mqtt.bambulab.com:8883"
	BambuMQTTCN    = "tls://cn.mqtt.bambulab.com:8883"
	BambuLocalPort = 8883 // Local MQTT port
)

// ConnectionMode represents the connection type
type ConnectionMode string

const (
	ModeLAN   ConnectionMode = "LAN"
	ModeCloud ConnectionMode = "CLOUD"
)

// PrintStage represents Bambu's stg_cur integer states
// Based on Python reverse-engineering: enum PrintStatus
type PrintStage int

const (
	StagePrinting         PrintStage = 0
	StageAutoBedLevel     PrintStage = 1
	StageHeatbedPreheat   PrintStage = 2
	StageCleaningNozzle   PrintStage = 3
	StageInspectingFirst  PrintStage = 4
	StageIdentifyingFirst PrintStage = 5
	StageIdle             PrintStage = 255
)

// PrintStageStrings maps integer stages to human-readable strings
var PrintStageStrings = map[PrintStage]string{
	0:   "Printing",
	1:   "Auto Bed Leveling",
	2:   "Heatbed Preheating",
	3:   "Cleaning Nozzle",
	4:   "Inspecting First Layer",
	5:   "Identifying First Layer",
	255: "Idle",
}

// String returns human-readable stage name
func (p PrintStage) String() string {
	if s, ok := PrintStageStrings[p]; ok {
		return s
	}
	return fmt.Sprintf("Unknown (%d)", p)
}

// Client represents a Bambu printer MQTT client
type Client struct {
	mu             sync.Mutex
	client         mqtt.Client
	mode           ConnectionMode
	region         string
	serial         string
	ipAddress      string
	accessCode     string
	username       string
	password       string
	connected      bool
	statusChan     chan PrinterStatus
	lastKnownState *BambuPrintPayload
	dumpFile       *os.File
}

// PrinterStatus represents current printer state (public API)
type PrinterStatus struct {
	State        string             `json:"state"`          // IDLE, PRINTING, PAUSED, COMPLETE, ERROR
	SubState     string             `json:"sub_state"`      // Detailed state like "Auto Bed Leveling"
	SubStateCode int                `json:"sub_state_code"` // stg_cur integer code
	Filename     string             `json:"filename"`
	Progress     float32            `json:"progress"`
	PrintTime    int64              `json:"print_time"`
	TimeLeft     int64              `json:"time_left"`
	Layer        int                `json:"layer"`
	TotalLayers  int                `json:"total_layers"`
	Temperatures map[string]float32 `json:"temperatures"`
	Targets      map[string]float32 `json:"targets"`
	FanSpeed     float32            `json:"fan_speed"`
	SpeedLevel   int                `json:"speed_level"`
	Online       bool               `json:"online"`
}

// BambuPrintPayload uses pointers for all fields to handle partial JSON diffs
// Based on Python reverse-engineering of Bambu Lab protocol
type BambuPrintPayload struct {
	Command          *string       `json:"command"`
	SequenceID       *string       `json:"sequence_id"`
	GcodeState       *string       `json:"gcode_state"`
	GcodeFile        *string       `json:"gcode_file"`
	Progress         *string       `json:"mc_percent"`
	Remaining        *string       `json:"mc_remaining_time"`
	Layer            *string       `json:"layer_num"`
	TotalLayers      *string       `json:"layer_total"`
	NozzleTemp       *float64      `json:"nozzle_temper"`
	BedTemp          *float64      `json:"bed_temper"`
	TargetNozzle     *float64      `json:"nozzle_target_temper"`
	TargetBed        *float64      `json:"bed_target_temper"`
	FanSpeed         *float64      `json:"cooling_fan_speed"`
	SpeedLevel       *int          `json:"spd_lvl"`
	PrintError       *int          `json:"print_error"`
	SubtaskName      *string       `json:"subtask_name"`
	StgCur           *int          `json:"stg_cur"` // Print stage integer
	Hms              []HmsItem     `json:"hms"`
	IPCam            *IPCamStatus  `json:"ipcam"`
	Upload           *UploadStatus `json:"upload"`
	Net              *NetStatus    `json:"net"`
	Ams              *AmsStatus    `json:"ams"`
	LightsReport     []LightReport `json:"lights_report"`
	XCam             *XCamStatus   `json:"xcam"`
	Online           *OnlineStatus `json:"online"`
	VtTray           *TrayInfo     `json:"vt_tray"`
	UpgradeState     *UpgradeState `json:"upgrade_state"`
	GcodeFilePrepare *string       `json:"gcode_file_prepare_percent"`
	PrintType        *string       `json:"print_type"`
	HomeFlag         *int          `json:"home_flag"`
	Sdcard           *bool         `json:"sdcard"`
}

// UpgradeState represents firmware upgrade state
type UpgradeState struct {
	SequenceID   *int    `json:"sequence_id"`
	Progress     *string `json:"progress"`
	Status       *string `json:"status"`
	ErrCode      *int    `json:"err_code"`
	ForceUpgrade *bool   `json:"force_upgrade"`
	Message      *string `json:"message"`
	Module       *string `json:"module"`
}

// HmsItem represents HMS error items
type HmsItem struct {
	Attr     *int `json:"attr"`
	Code     *int `json:"code"`
	Count    *int `json:"count"`
	Severity *int `json:"severity"`
}

// IPCamStatus represents camera status
type IPCamStatus struct {
	IPCamDev    *string `json:"ipcam_dev"`
	IPCamRecord *string `json:"ipcam_record"`
	Timelapse   *string `json:"timelapse"`
	Resolution  *string `json:"resolution"`
	TutkServer  *string `json:"tutk_server"`
	ModeBits    *int    `json:"mode_bits"`
}

// UploadStatus represents upload status
type UploadStatus struct {
	Status   *string `json:"status"`
	Progress *int    `json:"progress"`
	Message  *string `json:"message"`
}

// NetStatus represents network status
type NetStatus struct {
	Conf *int      `json:"conf"`
	Info []NetInfo `json:"info"`
}

// NetInfo represents network info
type NetInfo struct {
	IP   *int `json:"ip"`
	Mask *int `json:"mask"`
}

// AmsStatus represents AMS status
type AmsStatus struct {
	Ams              []AmsUnit `json:"ams"`
	AmsExistBits     *string   `json:"ams_exist_bits"`
	TrayExistBits    *string   `json:"tray_exist_bits"`
	TrayIsBblBits    *string   `json:"tray_is_bbl_bits"`
	TrayTar          *string   `json:"tray_tar"`
	TrayNow          *string   `json:"tray_now"`
	TrayPre          *string   `json:"tray_pre"`
	TrayReadDoneBits *string   `json:"tray_read_done_bits"`
	TrayReadingBits  *string   `json:"tray_reading_bits"`
	Version          *int      `json:"version"`
	InsertFlag       *bool     `json:"insert_flag"`
	PowerOnFlag      *bool     `json:"power_on_flag"`
}

// AmsUnit represents AMS unit
type AmsUnit struct {
	ID        *string    `json:"id"`
	Humidity  *string    `json:"humidity"`
	Temp      *string    `json:"temp"`
	Tray      []TrayInfo `json:"tray"`
	ChipID    *string    `json:"chip_id"`
	Check     *int       `json:"check"`
	DryTime   *int       `json:"dry_time"`
	Info      *string    `json:"info"`
	AmsExist  *bool      `json:"ams_exist"`
	TrayIsBbl *bool      `json:"tray_is_bbl"`
	TrayColor *string    `json:"tray_color"`
	TrayType  *string    `json:"tray_type"`
}

// TrayInfo represents filament tray info
type TrayInfo struct {
	ID            *string  `json:"id"`
	State         *int     `json:"state"`
	Remain        *int     `json:"remain"`
	K             *float64 `json:"k"`
	N             *int     `json:"n"`
	CaliIdx       *int     `json:"cali_idx"`
	TotalLen      *int     `json:"total_len"`
	TagUID        *string  `json:"tag_uid"`
	TrayIDName    *string  `json:"tray_id_name"`
	TrayInfoIdx   *string  `json:"tray_info_idx"`
	TrayType      *string  `json:"tray_type"`
	TraySubBrands *string  `json:"tray_sub_brands"`
	TrayColor     *string  `json:"tray_color"`
	TrayWeight    *string  `json:"tray_weight"`
	TrayDiameter  *string  `json:"tray_diameter"`
	TrayTemp      *string  `json:"tray_temp"`
	TrayTime      *string  `json:"tray_time"`
	BedTempType   *string  `json:"bed_temp_type"`
	BedTemp       *string  `json:"bed_temp"`
	NozzleTempMax *string  `json:"nozzle_temp_max"`
	NozzleTempMin *string  `json:"nozzle_temp_min"`
	XCamInfo      *string  `json:"xcam_info"`
	TrayUUID      *string  `json:"tray_uuid"`
	CTYPE         *int     `json:"ctype"`
	Cols          []string `json:"cols"`
}

// LightReport represents light status
type LightReport struct {
	Node *string `json:"node"`
	Mode *string `json:"mode"`
}

// XCamStatus represents XCam status
type XCamStatus struct {
	BuildplateMarkerDetector *bool `json:"buildplate_marker_detector"`
}

// OnlineStatus represents online status
type OnlineStatus struct {
	Ahb     *bool `json:"ahb"`
	Rfid    *bool `json:"rfid"`
	Version *int  `json:"version"`
}

// NewClient creates a new Bambu MQTT client
func NewClient() *Client {
	return &Client{
		statusChan:     make(chan PrinterStatus, 100),
		lastKnownState: &BambuPrintPayload{},
	}
}

// openDumpFile opens or reopens the dump file
func (c *Client) openDumpFile() error {
	if c.dumpFile != nil {
		c.dumpFile.Close()
	}

	f, err := os.OpenFile("bambu_mqtt_dump.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	c.dumpFile = f
	return nil
}

// dumpRawPacket writes raw MQTT payload to dump file with timestamp
func (c *Client) dumpRawPacket(payload []byte) {
	if c.dumpFile == nil {
		if err := c.openDumpFile(); err != nil {
			log.Printf("[BAMBU DUMP] Failed to open dump file: %v", err)
			return
		}
	}

	timestamp := time.Now().Format(time.RFC3339)
	line := fmt.Sprintf("[%s] RAW: %s\n", timestamp, string(payload))

	if _, err := c.dumpFile.WriteString(line); err != nil {
		log.Printf("[BAMBU DUMP] Failed to write to dump file: %v", err)
	}
}

// ConnectLAN connects to a Bambu printer via local MQTT
func (c *Client) ConnectLAN(ipAddress, accessCode, serial string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.mode = ModeLAN
	c.ipAddress = ipAddress
	c.accessCode = accessCode
	c.serial = serial

	log.Printf("[BAMBU MQTT] Connecting to LAN printer at %s (serial: %s)", ipAddress, serial)

	// MQTT broker URL for LAN
	brokerURL := fmt.Sprintf("tcps://%s:8883", ipAddress)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(fmt.Sprintf("nexus-hub-%s", serial))
	opts.SetUsername("bblp")
	opts.SetPassword(accessCode)
	opts.SetTLSConfig(&tls.Config{
		InsecureSkipVerify: true,
	})
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetConnectionLostHandler(c.connectionLostHandler)
	opts.SetOnConnectHandler(c.onConnectHandler)

	// Set message handler
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		c.handleMQTTMessage(msg.Payload())
	})

	c.client = mqtt.NewClient(opts)

	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to printer: %w", token.Error())
	}

	// Subscribe to report topic
	reportTopic := fmt.Sprintf("device/%s/report", serial)
	if token := c.client.Subscribe(reportTopic, 1, nil); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe: %w", token.Error())
	}

	c.connected = true
	log.Printf("[BAMBU MQTT] Connected to LAN printer: %s", serial)
	return nil
}

// onConnectHandler is called when connection is established
func (c *Client) onConnectHandler(client mqtt.Client) {
	log.Printf("[BAMBU MQTT] Connection established")

	// Send comprehensive pushall command (from Python lib)
	c.sendPushAllCommand()
}

// sendPushAllCommand sends the comprehensive pushall command
// Based on Python reverse-engineering: {"pushing": {"command": "pushall"}, "info": {"command": "get_version"}, "upgrade": {"command": "get_history"}}
func (c *Client) sendPushAllCommand() {
	if c.client == nil || !c.client.IsConnected() {
		return
	}

	requestTopic := fmt.Sprintf("device/%s/request", c.serial)

	// Send comprehensive pushall payload (exact from Python lib)
	payload := `{"pushing": {"command": "pushall"}, "info": {"command": "get_version"}, "upgrade": {"command": "get_history"}}`

	log.Printf("[BAMBU MQTT] Sending comprehensive pushall command to %s", requestTopic)

	if token := c.client.Publish(requestTopic, 1, false, payload); token.Wait() && token.Error() != nil {
		log.Printf("[BAMBU MQTT] Failed to send pushall: %v", token.Error())
	} else {
		log.Printf("[BAMBU MQTT] Pushall command sent successfully")
	}
}

// connectionLostHandler is called when connection is lost
func (c *Client) connectionLostHandler(client mqtt.Client, err error) {
	log.Printf("[BAMBU MQTT] Connection lost: %v", err)
	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()
}

// handleMQTTMessage processes incoming MQTT messages
func (c *Client) handleMQTTMessage(payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// DUMP RAW PACKET
	c.dumpRawPacket(payload)

	// Parse incoming payload
	var msg struct {
		Print *BambuPrintPayload `json:"print"`
	}

	if err := json.Unmarshal(payload, &msg); err != nil {
		log.Printf("[BAMBU MQTT] Failed to parse MQTT message: %v", err)
		return
	}

	if msg.Print == nil {
		return
	}

	// Merge incoming data into last known state (only non-nil fields)
	c.mergeState(msg.Print)

	// Convert to public PrinterStatus
	status := c.convertToStatus()

	// Send to channel (non-blocking)
	select {
	case c.statusChan <- status:
	default:
		// Channel full, skip
	}
}

// mergeState safely merges non-nil pointers from incoming into lastKnownState
// This prevents zeroing out fields when Bambu sends partial JSON diffs
func (c *Client) mergeState(incoming *BambuPrintPayload) {
	last := c.lastKnownState

	// String pointers
	if incoming.Command != nil {
		last.Command = incoming.Command
	}
	if incoming.SequenceID != nil {
		last.SequenceID = incoming.SequenceID
	}
	if incoming.GcodeState != nil {
		last.GcodeState = incoming.GcodeState
	}
	if incoming.GcodeFile != nil {
		last.GcodeFile = incoming.GcodeFile
	}
	if incoming.Progress != nil {
		last.Progress = incoming.Progress
	}
	if incoming.Remaining != nil {
		last.Remaining = incoming.Remaining
	}
	if incoming.Layer != nil {
		last.Layer = incoming.Layer
	}
	if incoming.TotalLayers != nil {
		last.TotalLayers = incoming.TotalLayers
	}
	if incoming.SubtaskName != nil {
		last.SubtaskName = incoming.SubtaskName
	}
	if incoming.GcodeFilePrepare != nil {
		last.GcodeFilePrepare = incoming.GcodeFilePrepare
	}
	if incoming.PrintType != nil {
		last.PrintType = incoming.PrintType
	}

	// Float64 pointers (temperatures, fan speed, progress)
	if incoming.NozzleTemp != nil {
		last.NozzleTemp = incoming.NozzleTemp
	}
	if incoming.BedTemp != nil {
		last.BedTemp = incoming.BedTemp
	}
	if incoming.TargetNozzle != nil {
		last.TargetNozzle = incoming.TargetNozzle
	}
	if incoming.TargetBed != nil {
		last.TargetBed = incoming.TargetBed
	}
	if incoming.FanSpeed != nil {
		last.FanSpeed = incoming.FanSpeed
	}

	// Int pointers
	if incoming.SpeedLevel != nil {
		last.SpeedLevel = incoming.SpeedLevel
	}
	if incoming.PrintError != nil {
		last.PrintError = incoming.PrintError
	}
	if incoming.StgCur != nil {
		last.StgCur = incoming.StgCur
	}
	if incoming.HomeFlag != nil {
		last.HomeFlag = incoming.HomeFlag
	}

	// Bool pointers
	if incoming.Sdcard != nil {
		last.Sdcard = incoming.Sdcard
	}

	// Slices
	if len(incoming.Hms) > 0 {
		last.Hms = incoming.Hms
	}
	if len(incoming.LightsReport) > 0 {
		last.LightsReport = incoming.LightsReport
	}

	// Structs
	if incoming.IPCam != nil {
		last.IPCam = incoming.IPCam
	}
	if incoming.Upload != nil {
		last.Upload = incoming.Upload
	}
	if incoming.Net != nil {
		last.Net = incoming.Net
	}
	if incoming.Ams != nil {
		last.Ams = incoming.Ams
	}
	if incoming.XCam != nil {
		last.XCam = incoming.XCam
	}
	if incoming.Online != nil {
		last.Online = incoming.Online
	}
	if incoming.VtTray != nil {
		last.VtTray = incoming.VtTray
	}
	if incoming.UpgradeState != nil {
		last.UpgradeState = incoming.UpgradeState
	}
}

// convertToStatus converts lastKnownState to public PrinterStatus
func (c *Client) convertToStatus() PrinterStatus {
	last := c.lastKnownState

	status := PrinterStatus{
		Temperatures: make(map[string]float32),
		Targets:      make(map[string]float32),
		Online:       true,
	}

	// Map state correctly using gcode_state
	status.State = c.mapState(last.GcodeState)

	// Map sub-state using stg_cur integer
	if last.StgCur != nil {
		status.SubStateCode = *last.StgCur
		status.SubState = PrintStage(*last.StgCur).String()
	} else {
		status.SubStateCode = 255
		status.SubState = "Idle"
	}

	// Parse string fields safely
	if last.GcodeFile != nil {
		status.Filename = *last.GcodeFile
	}

	if last.Progress != nil {
		if val, err := parseFloat(*last.Progress); err == nil {
			status.Progress = float32(val)
		}
	}

	if last.Remaining != nil {
		if val, err := parseFloat(*last.Remaining); err == nil {
			status.TimeLeft = int64(val)
		}
	}

	if last.Layer != nil {
		if val, err := parseInt(*last.Layer); err == nil {
			status.Layer = val
		}
	}

	if last.TotalLayers != nil {
		if val, err := parseInt(*last.TotalLayers); err == nil {
			status.TotalLayers = val
		}
	}

	// Temperatures (float64 pointers)
	if last.NozzleTemp != nil {
		status.Temperatures["nozzle"] = float32(*last.NozzleTemp)
	}
	if last.BedTemp != nil {
		status.Temperatures["bed"] = float32(*last.BedTemp)
	}
	if last.TargetNozzle != nil {
		status.Targets["nozzle"] = float32(*last.TargetNozzle)
	}
	if last.TargetBed != nil {
		status.Targets["bed"] = float32(*last.TargetBed)
	}

	// Fan speed
	if last.FanSpeed != nil {
		status.FanSpeed = float32(*last.FanSpeed)
	}

	// Speed level
	if last.SpeedLevel != nil {
		status.SpeedLevel = *last.SpeedLevel
	}

	return status
}

// mapState maps Bambu's proprietary gcode_state strings to our internal states
func (c *Client) mapState(state *string) string {
	if state == nil {
		return "IDLE"
	}

	s := strings.ToUpper(*state)

	switch s {
	case "RUNNING", "PREPARE":
		return "PRINTING"
	case "PAUSE", "PAUSED":
		return "PAUSED"
	case "FINISH", "FINISHED":
		return "COMPLETE"
	case "FAILED", "ERROR":
		return "ERROR"
	case "IDLE":
		return "IDLE"
	default:
		// Check if it contains common keywords
		if strings.Contains(s, "RUN") || strings.Contains(s, "PRINT") {
			return "PRINTING"
		}
		if strings.Contains(s, "PAUSE") {
			return "PAUSED"
		}
		if strings.Contains(s, "FINISH") || strings.Contains(s, "COMPLETE") {
			return "COMPLETE"
		}
		return "IDLE"
	}
}

// GetStatus returns current printer status
func (c *Client) GetStatus() PrinterStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lastKnownState == nil {
		return PrinterStatus{
			State:        "IDLE",
			SubState:     "Idle",
			SubStateCode: 255,
			Temperatures: make(map[string]float32),
			Targets:      make(map[string]float32),
		}
	}

	return c.convertToStatus()
}

// IsPrinting returns true if printer is currently printing or paused
func (c *Client) IsPrinting() bool {
	status := c.GetStatus()
	return status.State == "PRINTING" || status.State == "PAUSED"
}

// GetStatusChan returns the status channel for real-time updates
func (c *Client) GetStatusChan() <-chan PrinterStatus {
	return c.statusChan
}

// IsConnected returns connection status
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// Close disconnects the client
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.dumpFile != nil {
		c.dumpFile.Close()
		c.dumpFile = nil
	}

	if c.client != nil {
		c.client.Disconnect(250)
	}
	c.connected = false
	close(c.statusChan)
	log.Printf("[BAMBU MQTT] Client disconnected")
}

// SendGCode sends a G-code command with hardware safety check
// Returns error if printer is currently printing or paused
func (c *Client) SendGCode(gcode string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// HARDWARE SAFETY LOCK: Do not send G-code if printing
	if c.lastKnownState != nil && c.lastKnownState.GcodeState != nil {
		state := strings.ToUpper(*c.lastKnownState.GcodeState)
		if state == "RUNNING" || state == "PAUSE" || state == "PAUSED" {
			return fmt.Errorf("hardware safety lock: cannot send G-code while printing (state: %s)", state)
		}
	}

	if !c.connected || c.client == nil {
		return fmt.Errorf("not connected")
	}

	requestTopic := fmt.Sprintf("device/%s/request", c.serial)
	// Exact payload from Python lib: sequence_id "0" for G-code commands
	payload := fmt.Sprintf(`{"print": {"sequence_id": "0", "command": "gcode_line", "param": "%s \n"}}`, gcode)

	if token := c.client.Publish(requestTopic, 1, false, payload); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to send G-code: %w", token.Error())
	}

	return nil
}

// Pause sends pause command (no sequence_id for system commands)
func (c *Client) Pause() error {
	return c.sendSystemCommand("pause")
}

// Resume sends resume command (no sequence_id for system commands)
func (c *Client) Resume() error {
	return c.sendSystemCommand("resume")
}

// Stop sends stop command (no sequence_id for system commands)
func (c *Client) Stop() error {
	return c.sendSystemCommand("stop")
}

// sendSystemCommand sends system commands without sequence_id
// Exact payloads from Python reverse-engineering
func (c *Client) sendSystemCommand(command string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.client == nil {
		return fmt.Errorf("not connected")
	}

	requestTopic := fmt.Sprintf("device/%s/request", c.serial)
	// Exact payload from Python lib: NO sequence_id for system commands
	payload := fmt.Sprintf(`{"print": {"command": "%s"}}`, command)

	if token := c.client.Publish(requestTopic, 1, false, payload); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to send command: %w", token.Error())
	}

	return nil
}

// parseFloat safely parses a string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// parseInt safely parses a string to int
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// GetLoginURL returns the Bambu Lab login URL for OAuth-style flow
func GetLoginURL() string {
	return "https://bambulab.com/en/login"
}
