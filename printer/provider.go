// Package printer provides a unified interface for multi-vendor 3D printer support
package printer

import (
	"context"
	"time"
)

// PrinterType represents the vendor/type of printer
type PrinterType string

const (
	PrinterTypeMoonraker PrinterType = "MOONRAKER"
	PrinterTypeBambuLab  PrinterType = "BAMBULAB"
	PrinterTypePrusa     PrinterType = "PRUSA"
	PrinterTypeGeneric   PrinterType = "GENERIC"
)

// PrinterStatus represents the current state of any printer
type PrinterStatus struct {
	PrinterID    int64
	State        string             // idle, printing, paused, error, complete
	Filename     string             // Current file being printed
	Progress     float32            // 0-100 percentage
	PrintTime    int64              // Seconds elapsed
	TimeLeft     int64              // Seconds remaining
	Layer        int                // Current layer
	TotalLayers  int                // Total layers
	Temperatures map[string]float32 // Component -> temperature (nozzle, bed, chamber)
	Targets      map[string]float32 // Component -> target temperature
	FanSpeed     float32            // 0-100 percentage
	SpeedLevel   int                // Print speed level (1-5)
	Capabilities PrinterCapabilities
	VendorData   map[string]interface{} // Vendor-specific data
}

// PrinterCapabilities describes what features a printer supports
type PrinterCapabilities struct {
	HasCamera       bool
	HasLEDControl   bool
	HasFanControl   bool
	HasSpeedControl bool
	HasLaser        bool
	MaxExtruders    int
	SupportedGCode  []string
}

// PrintJob represents a job being sent to a printer
type PrintJob struct {
	Filename     string
	FileData     []byte // Or URL for cloud printers
	FilamentType string
	PrinterModel string
	Metadata     map[string]interface{}
}

// PrinterCommand represents a command to send to a printer
type PrinterCommand struct {
	Type   string                 // pause, resume, stop, estop, set_temp, set_fan, etc.
	Params map[string]interface{} // Command-specific parameters
}

// PrinterProvider is the interface that all vendor implementations must satisfy
type PrinterProvider interface {
	// Connect establishes connection to the printer
	Connect(ctx context.Context) error

	// Disconnect closes the connection
	Disconnect() error

	// IsConnected returns connection status
	IsConnected() bool

	// GetStatus fetches current printer status
	GetStatus(ctx context.Context) (*PrinterStatus, error)

	// SendCommand sends a control command
	SendCommand(ctx context.Context, cmd PrinterCommand) error

	// UploadAndPrint uploads a file and starts printing
	UploadAndPrint(ctx context.Context, job PrintJob) error

	// GetFileInfo retrieves information about a file on the printer
	GetFileInfo(ctx context.Context, filepath string) (map[string]interface{}, error)

	// ListFiles returns available files on the printer
	ListFiles(ctx context.Context) ([]string, error)

	// GetCapabilities returns printer capabilities
	GetCapabilities() PrinterCapabilities
}

// PrinterConfig holds configuration for connecting to a printer
type PrinterConfig struct {
	PrinterID   int64
	PrinterType PrinterType
	Address     string // IP address or hostname
	Port        int
	Username    string            // For cloud services
	Password    string            // For cloud services
	AccessToken string            // For API authentication
	Secret      string            // For cloud services (BambuLab)
	Extra       map[string]string // Vendor-specific config
}

// Event represents a printer event for subscription
type PrinterEvent struct {
	PrinterID int64
	Type      string // status_change, file_complete, error, etc.
	Timestamp time.Time
	Data      interface{}
}

// EventHandler is a function that handles printer events
type EventHandler func(event PrinterEvent)
