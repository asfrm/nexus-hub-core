// Package validator provides input validation helpers
package validator

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

var (
	accessCodeRegex = regexp.MustCompile(`^[A-Z0-9]{8}$`)
	serialRegex     = regexp.MustCompile(`^[A-Z0-9]{12,16}$`)
)

// ValidateAccessCode validates Bambu Lab access code (8 alphanumeric characters)
func ValidateAccessCode(code string) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if !accessCodeRegex.MatchString(code) {
		return fmt.Errorf("access code must be exactly 8 alphanumeric characters (A-Z, 0-9)")
	}
	return nil
}

// ValidateSerialNumber validates Bambu Lab serial number
func ValidateSerialNumber(serial string) error {
	serial = strings.ToUpper(strings.TrimSpace(serial))
	if !serialRegex.MatchString(serial) {
		return fmt.Errorf("serial number must be 12-16 alphanumeric characters")
	}
	return nil
}

// ValidateIPAddress validates an IP address (IPv4 or IPv6)
func ValidateIPAddress(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	return nil
}

// ValidateHostname validates a hostname
func ValidateHostname(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}
	if len(hostname) > 253 {
		return fmt.Errorf("hostname too long (max 253 characters)")
	}
	// Simple hostname validation
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if !hostnameRegex.MatchString(hostname) {
		return fmt.Errorf("invalid hostname format")
	}
	return nil
}

// ValidateClusterName validates cluster name
func ValidateClusterName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("cluster name cannot be empty")
	}
	if len(name) > 50 {
		return fmt.Errorf("cluster name too long (max 50 characters)")
	}
	return nil
}

// ValidatePrinterName validates printer name
func ValidatePrinterName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("printer name cannot be empty")
	}
	if len(name) > 50 {
		return fmt.Errorf("printer name too long (max 50 characters)")
	}
	return nil
}

// ValidateFilamentType validates filament type
func ValidateFilamentType(filament string) error {
	validTypes := map[string]bool{
		"PLA": true, "PETG": true, "ABS": true, "ASA": true,
		"TPU": true, "PC": true, "Nylon": true, "PVA": true,
		"HIPS": true, "PP": true, "Unknown": true,
	}

	filament = strings.TrimSpace(filament)
	if filament == "" {
		return nil // Optional field
	}

	if !validTypes[filament] {
		return fmt.Errorf("unknown filament type: %s", filament)
	}
	return nil
}

// SanitizeString removes potentially dangerous characters
func SanitizeString(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")
	// Trim whitespace
	s = strings.TrimSpace(s)
	// Limit length
	if len(s) > 500 {
		s = s[:500]
	}
	return s
}
