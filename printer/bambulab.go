// Package printer provides integration with Bambu Lab cloud and local APIs
package printer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// BambuLab API endpoints
const (
	BambuCloudAPIUS    = "https://api.bambulab.com"
	BambuCloudAPIEU    = "https://api.bambulab.eu"
	BambuCloudAPICN    = "https://api.bambulab.cn"
	BambuMQTTBrokerUS  = "us.mqtt.bambulab.com"
	BambuMQTTBrokerEU  = "eu.mqtt.bambulab.com"
	BambuMQTTBrokerCN  = "cn.mqtt.bambulab.com"
	BambuLocalPort     = 8883 // Local MQTT port
	BambuLocalHTTPPort = 8890 // Local HTTP API port
)

// BambuConnectionMode represents the connection type
type BambuConnectionMode string

const (
	BambuModeCloud BambuConnectionMode = "CLOUD"
	BambuModeLAN   BambuConnectionMode = "LAN"
)

// BambuProvider implements printer.Provider for BambuLab printers
type BambuProvider struct {
	config     PrinterConfig
	httpClient *http.Client
	mu         sync.Mutex
	connected  bool
	mode       BambuConnectionMode

	// Authentication
	accessToken  string
	tokenExpiry  time.Time
	refreshToken string

	// Printer info
	printerSerial string
	printerRegion string

	// 2FA
	tfaRequired bool
	tfaTicket   string

	// Event handlers
	eventHandlers []EventHandler

	// Last known status
	lastStatus *PrinterStatus
}

// BambuAuthRequest is the login request payload
type BambuAuthRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

// BambuAuthResponse is the response from Bambu cloud login
type BambuAuthResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	ErrorCode    int    `json:"error_code"`
	Message      string `json:"message"`
	TFATicket    string `json:"tfa_ticket"`
}

// BambuPrinterInfo contains printer information from cloud
type BambuPrinterInfo struct {
	Serial    string `json:"serial"`
	Name      string `json:"name"`
	Model     string `json:"model"`
	Online    bool   `json:"online"`
	IPAddress string `json:"ip"`
	BindCode  string `json:"bind_code"`
}

// BambuPrinter represents a printer discovered from cloud account
type BambuPrinter struct {
	Serial    string `json:"serial"`
	Name      string `json:"name"`
	Model     string `json:"model"`
	Online    bool   `json:"online"`
	IPAddress string `json:"ip"`
	BindCode  string `json:"bind_code,omitempty"`
}

// BambuStatusResponse is the MQTT/HTTP status response
type BambuStatusResponse struct {
	Print struct {
		SubtaskID    string `json:"subtask_id"`
		State        int    `json:"gcode_state"`
		Progress     string `json:"mc_percent"`
		Remaining    string `json:"mc_remaining_time"`
		Layer        string `json:"layer_num"`
		TotalLayers  string `json:"layer_total"`
		File         string `json:"gcode_file"`
		NozzleTemp   string `json:"nozzle_temper"`
		BedTemp      string `json:"bed_temper"`
		TargetNozzle string `json:"nozzle_target_temper"`
		TargetBed    string `json:"bed_target_temper"`
		FanSpeed     string `json:"cooling_fan_speed"`
		SpeedLevel   int    `json:"spd_lvl"`
	} `json:"print"`
}

// NewBambuProvider creates a new BambuLab printer provider
func NewBambuProvider(config PrinterConfig) *BambuProvider {
	mode := BambuModeLAN
	if config.AccessToken != "" || config.Username != "" {
		mode = BambuModeCloud
	}

	// Parse extra config for mode
	if config.Extra != nil {
		if m, ok := config.Extra["mode"]; ok {
			if m == "LAN" {
				mode = BambuModeLAN
			} else if m == "CLOUD" {
				mode = BambuModeCloud
			}
		}
	}

	return &BambuProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
		},
		mode:          mode,
		eventHandlers: make([]EventHandler, 0),
		printerSerial: config.Extra["serial"],
		printerRegion: config.Extra["region"],
	}
}

// getCloudAPIURL returns the appropriate cloud API URL based on region
func (b *BambuProvider) getCloudAPIURL() string {
	region := strings.ToUpper(b.printerRegion)
	switch region {
	case "EU":
		return BambuCloudAPIEU
	case "CN":
		return BambuCloudAPICN
	default:
		return BambuCloudAPIUS
	}
}

// getMQTTBroker returns the appropriate MQTT broker based on region
func (b *BambuProvider) getMQTTBroker() string {
	region := strings.ToUpper(b.printerRegion)
	switch region {
	case "EU":
		return BambuMQTTBrokerEU
	case "CN":
		return BambuMQTTBrokerCN
	default:
		return BambuMQTTBrokerUS
	}
}

// logDebug logs debug messages with context
func (b *BambuProvider) logDebug(format string, args ...interface{}) {
	log.Printf("[BAMBU:%s] "+format, append([]interface{}{b.config.Extra["serial"]}, args...)...)
}

// Connect establishes connection to BambuLab printer
func (b *BambuProvider) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.connected {
		b.logDebug("Already connected, skipping")
		return nil
	}

	b.logDebug("Connecting in %s mode", b.mode)

	if b.mode == BambuModeLAN {
		// LAN mode - just test connection
		if b.config.Extra["local_ip"] == "" {
			return fmt.Errorf("LAN mode requires local_ip in config")
		}
		if b.config.Extra["access_code"] == "" {
			return fmt.Errorf("LAN mode requires access_code in config")
		}

		// Test connection
		_, err := b.getLocalStatus(ctx)
		if err != nil {
			return fmt.Errorf("LAN connection test failed: %w", err)
		}
		b.logDebug("LAN connection successful")
	} else {
		// Cloud mode - authenticate
		if err := b.authenticate(ctx); err != nil {
			return fmt.Errorf("cloud authentication failed: %w", err)
		}

		// Get printer info if serial not provided
		if b.printerSerial == "" {
			if err := b.getPrinterInfo(ctx); err != nil {
				return fmt.Errorf("failed to get printer info: %w", err)
			}
		}
		b.logDebug("Cloud authentication successful for printer %s", b.printerSerial)
	}

	b.connected = true
	return nil
}

// Authenticate logs into Bambu cloud and gets access token (public method)
func (b *BambuProvider) Authenticate(ctx context.Context) error {
	return b.authenticate(ctx)
}

// authenticate logs into Bambu cloud and gets access token (internal)
func (b *BambuProvider) authenticate(ctx context.Context) error {
	// Check if we have a valid token
	if b.accessToken != "" && time.Now().Before(b.tokenExpiry) {
		b.logDebug("Using cached access token (expires in %v)", time.Until(b.tokenExpiry))
		return nil
	}

	// Use provided access token
	if b.config.AccessToken != "" {
		b.accessToken = b.config.AccessToken
		b.tokenExpiry = time.Now().Add(24 * time.Hour)
		b.logDebug("Using provided access token")
		return nil
	}

	// Username/password authentication
	if b.config.Username == "" {
		return fmt.Errorf("no credentials available for cloud authentication")
	}

	b.logDebug("Authenticating with username/password for user: %s", b.config.Username)

	authURL := fmt.Sprintf("%s/v1/user-service/user/login", b.getCloudAPIURL())

	data := url.Values{}
	data.Set("account", b.config.Username)
	data.Set("password", b.config.Password)

	req, err := http.NewRequestWithContext(ctx, "POST", authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	b.logDebug("Sending auth request to: %s", authURL)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	b.logDebug("Auth response status: %d", resp.StatusCode)
	b.logDebug("Auth response body (masked): %s", maskSensitiveData(string(body)))

	var authResp BambuAuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return fmt.Errorf("failed to parse auth response: %w", err)
	}

	// Check for 2FA requirement
	if authResp.TFATicket != "" {
		b.logDebug("2FA required, ticket: %s", authResp.TFATicket[:8]+"...")
		b.tfaRequired = true
		b.tfaTicket = authResp.TFATicket
		return fmt.Errorf("2FA verification required")
	}

	if authResp.AccessToken == "" {
		return fmt.Errorf("authentication failed: %s", authResp.Message)
	}

	b.accessToken = authResp.AccessToken
	b.refreshToken = authResp.RefreshToken
	b.tokenExpiry = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)
	b.tfaRequired = false

	b.logDebug("Authentication successful, token expires in: %v", time.Until(b.tokenExpiry))
	return nil
}

// Verify2FA completes 2FA verification
func (b *BambuProvider) Verify2FA(ctx context.Context, code string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.tfaRequired || b.tfaTicket == "" {
		return fmt.Errorf("2FA not required")
	}

	b.logDebug("Verifying 2FA code for ticket: %s", b.tfaTicket[:8]+"...")

	authURL := fmt.Sprintf("%s/v1/user-service/user/login/2fa", b.getCloudAPIURL())

	data := url.Values{}
	data.Set("code", code)
	data.Set("tfa_ticket", b.tfaTicket)

	req, err := http.NewRequestWithContext(ctx, "POST", authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	b.logDebug("2FA response status: %d", resp.StatusCode)
	b.logDebug("2FA response body: %s", string(body))

	var authResp BambuAuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return err
	}

	if authResp.AccessToken == "" {
		return fmt.Errorf("2FA verification failed: %s", authResp.Message)
	}

	b.accessToken = authResp.AccessToken
	b.refreshToken = authResp.RefreshToken
	b.tokenExpiry = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)
	b.tfaRequired = false
	b.tfaTicket = ""

	b.logDebug("2FA verification successful")
	return nil
}

// getPrinterInfo fetches printer information from cloud
func (b *BambuProvider) getPrinterInfo(ctx context.Context) error {
	apiURL := fmt.Sprintf("%s/v1/iot-service/api/user/bind", b.getCloudAPIURL())

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.accessToken)

	b.logDebug("Fetching printer list from: %s", apiURL)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	b.logDebug("Printer list response: %s", string(body))

	var printers []BambuPrinterInfo
	if err := json.Unmarshal(body, &printers); err != nil {
		return err
	}

	b.logDebug("Found %d printers in account", len(printers))

	// Find matching printer by serial or use first one
	for _, p := range printers {
		b.logDebug("Printer: %s (%s) - Online: %v", p.Name, p.Serial, p.Online)
		if p.Serial == b.printerSerial || b.printerSerial == "" {
			b.printerSerial = p.Serial
			b.logDebug("Selected printer: %s", b.printerSerial)
			break
		}
	}

	if b.printerSerial == "" {
		return fmt.Errorf("no printer found in account")
	}

	return nil
}

// DiscoverPrinters returns all printers bound to the cloud account
func (b *BambuProvider) DiscoverPrinters(ctx context.Context) ([]BambuPrinter, error) {
	if err := b.authenticate(ctx); err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/v1/iot-service/api/user/bind", b.getCloudAPIURL())

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.accessToken)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var printers []BambuPrinterInfo
	if err := json.Unmarshal(body, &printers); err != nil {
		return nil, err
	}

	result := make([]BambuPrinter, len(printers))
	for i, p := range printers {
		result[i] = BambuPrinter{
			Serial:    p.Serial,
			Name:      p.Name,
			Model:     p.Model,
			Online:    p.Online,
			IPAddress: p.IPAddress,
			BindCode:  p.BindCode,
		}
	}

	return result, nil
}

// Disconnect closes the connection
func (b *BambuProvider) Disconnect() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.connected = false
	b.logDebug("Disconnected")
	return nil
}

// IsConnected returns connection status
func (b *BambuProvider) IsConnected() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.connected
}

// GetStatus fetches current printer status
func (b *BambuProvider) GetStatus(ctx context.Context) (*PrinterStatus, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Try local API first if in LAN mode or local_ip is configured
	if b.mode == BambuModeLAN || b.config.Extra["local_ip"] != "" {
		if status, err := b.getLocalStatus(ctx); err == nil {
			b.lastStatus = status
			return status, nil
		}
	}

	// Fall back to cloud API
	return b.getCloudStatus(ctx)
}

// getLocalStatus fetches status from local network HTTP API
func (b *BambuProvider) getLocalStatus(ctx context.Context) (*PrinterStatus, error) {
	localIP := b.config.Extra["local_ip"]
	accessCode := b.config.Extra["access_code"]

	apiURL := fmt.Sprintf("https://%s:%d/api/v1/task", localIP, BambuLocalHTTPPort)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("access_code", accessCode)

	b.logDebug("Fetching local status from: %s", apiURL)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	b.logDebug("Local status response: %s", string(body))

	var statusResp BambuStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, err
	}

	return b.parseStatusResponse(&statusResp), nil
}

// getCloudStatus fetches status from Bambu cloud
func (b *BambuProvider) getCloudStatus(ctx context.Context) (*PrinterStatus, error) {
	if b.printerSerial == "" {
		return nil, fmt.Errorf("printer serial not available")
	}

	apiURL := fmt.Sprintf("%s/v1/iot-service/api/user/bind/printer/%s/task", b.getCloudAPIURL(), b.printerSerial)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.accessToken)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var statusResp BambuStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, err
	}

	return b.parseStatusResponse(&statusResp), nil
}

// parseStatusResponse converts Bambu API response to unified status
func (b *BambuProvider) parseStatusResponse(resp *BambuStatusResponse) *PrinterStatus {
	state := "idle"
	switch resp.Print.State {
	case 0:
		state = "idle"
	case 1:
		state = "printing"
	case 2:
		state = "paused"
	case 3:
		state = "error"
	}

	progress := 0.0
	fmt.Sscanf(resp.Print.Progress, "%f", &progress)

	timeLeft := int64(0)
	fmt.Sscanf(resp.Print.Remaining, "%d", &timeLeft)

	layer := 0
	fmt.Sscanf(resp.Print.Layer, "%d", &layer)

	totalLayers := 0
	fmt.Sscanf(resp.Print.TotalLayers, "%d", &totalLayers)

	nozzleTemp := 0.0
	fmt.Sscanf(resp.Print.NozzleTemp, "%f", &nozzleTemp)

	bedTemp := 0.0
	fmt.Sscanf(resp.Print.BedTemp, "%f", &bedTemp)

	targetNozzle := 0.0
	fmt.Sscanf(resp.Print.TargetNozzle, "%f", &targetNozzle)

	targetBed := 0.0
	fmt.Sscanf(resp.Print.TargetBed, "%f", &targetBed)

	fanSpeed := 0.0
	fmt.Sscanf(resp.Print.FanSpeed, "%f", &fanSpeed)

	// Calculate print time from remaining
	printTime := int64(0)
	if timeLeft > 0 && progress > 0 {
		printTime = int64(float64(timeLeft)/(1-progress/100) - float64(timeLeft))
	}

	return &PrinterStatus{
		PrinterID:   b.config.PrinterID,
		State:       state,
		Filename:    resp.Print.File,
		Progress:    float32(progress),
		PrintTime:   printTime,
		TimeLeft:    timeLeft,
		Layer:       layer,
		TotalLayers: totalLayers,
		Temperatures: map[string]float32{
			"nozzle": float32(nozzleTemp),
			"bed":    float32(bedTemp),
		},
		Targets: map[string]float32{
			"nozzle": float32(targetNozzle),
			"bed":    float32(targetBed),
		},
		FanSpeed: float32(fanSpeed),
		Capabilities: PrinterCapabilities{
			HasCamera:       true,
			HasLEDControl:   true,
			HasFanControl:   true,
			HasSpeedControl: true,
			HasLaser:        false,
			MaxExtruders:    1,
		},
		VendorData: map[string]interface{}{
			"subtask_id":  resp.Print.SubtaskID,
			"speed_level": resp.Print.SpeedLevel,
		},
	}
}

// SendCommand sends a control command to the printer
func (b *BambuProvider) SendCommand(ctx context.Context, cmd PrinterCommand) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Use local API for commands (more reliable)
	localIP := b.config.Extra["local_ip"]
	accessCode := b.config.Extra["access_code"]

	if localIP == "" && b.mode == BambuModeCloud {
		// For cloud mode without local IP, use cloud commands
		return b.sendCloudCommand(ctx, cmd)
	}

	if localIP == "" {
		return fmt.Errorf("local IP not configured, commands require local connection")
	}

	return b.sendLocalCommand(ctx, localIP, accessCode, cmd)
}

// sendLocalCommand sends a command via local HTTP API
func (b *BambuProvider) sendLocalCommand(ctx context.Context, localIP, accessCode string, cmd PrinterCommand) error {
	apiURL := fmt.Sprintf("https://%s:%d/api/v1/print-control", localIP, BambuLocalHTTPPort)

	command := map[string]interface{}{
		"command": cmd.Type,
	}

	// Map command types
	switch cmd.Type {
	case "pause":
		command["command"] = "gcode_pause"
	case "resume":
		command["command"] = "gcode_resume"
	case "stop":
		command["command"] = "gcode_stop"
	case "set_fan":
		command["command"] = "gcode"
		speed := 0
		if s, ok := cmd.Params["speed"].(float64); ok {
			speed = int(s * 255 / 100)
		}
		command["param"] = fmt.Sprintf("M106 S%d", speed)
	case "set_speed":
		command["command"] = "print_speed"
		if level, ok := cmd.Params["level"].(int); ok {
			command["param"] = level
		}
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return err
	}

	b.logDebug("Sending local command: %s", string(jsonData))

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_code", accessCode)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	b.logDebug("Command response: %d - %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("command failed: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendCloudCommand sends a command via cloud API
func (b *BambuProvider) sendCloudCommand(ctx context.Context, cmd PrinterCommand) error {
	if b.printerSerial == "" {
		return fmt.Errorf("printer serial not available")
	}

	apiURL := fmt.Sprintf("%s/v1/iot-service/api/user/bind/printer/%s/command", b.getCloudAPIURL(), b.printerSerial)

	command := map[string]interface{}{
		"command": cmd.Type,
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.accessToken)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloud command failed: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// UploadAndPrint uploads a file and starts printing (LAN mode only)
func (b *BambuProvider) UploadAndPrint(ctx context.Context, job PrintJob) error {
	return fmt.Errorf("file upload not implemented for BambuLab - use Moonraker printers for upload or send G-Code via Bambu Studio")
}

// GetFileInfo retrieves information about a file
func (b *BambuProvider) GetFileInfo(ctx context.Context, filepath string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListFiles returns available files on the printer
func (b *BambuProvider) ListFiles(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

// GetCapabilities returns printer capabilities
func (b *BambuProvider) GetCapabilities() PrinterCapabilities {
	return PrinterCapabilities{
		HasCamera:       true,
		HasLEDControl:   true,
		HasFanControl:   true,
		HasSpeedControl: true,
		HasLaser:        false,
		MaxExtruders:    1,
	}
}

// IsTFAPending returns true if 2FA verification is required
func (b *BambuProvider) IsTFAPending() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tfaRequired
}

// GetTFATicket returns the current 2FA ticket
func (b *BambuProvider) GetTFATicket() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tfaTicket
}

// GetMode returns the current connection mode
func (b *BambuProvider) GetMode() BambuConnectionMode {
	return b.mode
}

// OnEvent registers an event handler
func (b *BambuProvider) OnEvent(handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.eventHandlers = append(b.eventHandlers, handler)
}

// emitEvent sends an event to all registered handlers
func (b *BambuProvider) emitEvent(event PrinterEvent) {
	for _, handler := range b.eventHandlers {
		go handler(event)
	}
}

// maskSensitiveData masks passwords and tokens in log output
func maskSensitiveData(s string) string {
	masks := []string{"password", "access_token", "refresh_token", "secret"}
	result := s
	for _, mask := range masks {
		result = strings.ReplaceAll(result, mask, "***REDACTED***")
	}
	return result
}

// EncodeToBase64 encodes a string to base64
func EncodeToBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// DecodeFromBase64 decodes a base64 string
func DecodeFromBase64(s string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
