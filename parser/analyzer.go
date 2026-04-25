// Package parser provides 3MF and GCODE file analysis
package parser

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ThreeMFMetadata represents 3MF slicing metadata
type ThreeMFMetadata struct {
	Slicing struct {
		Plate []struct {
			PrintTime      int64   `xml:"print_time,attr"`
			FilamentUsed   float32 `xml:"filament_used,attr"`
			FilamentWeight float32 `xml:"filament_weight,attr"`
			LayerHeight    float32 `xml:"layer_height,attr"`
			Infill         string  `xml:"infill,attr"`
			Supports       bool    `xml:"supports,attr"`
			PlateType      string  `xml:"plate_type,attr"`
			NozzleTemper   float32 `xml:"nozzle_temper,attr"`
			BedTemper      float32 `xml:"bed_temper,attr"`
		} `xml:"plate"`
	} `xml:"slicing"`
	Materials []struct {
		ID     string  `xml:"id,attr"`
		Name   string  `xml:"name,attr"`
		Color  string  `xml:"color,attr"`
		Type   string  `xml:"type,attr"`
		Weight float32 `xml:"weight,attr"`
	} `xml:"materials"`
}

// AnalysisResult contains parsed file information
type AnalysisResult struct {
	Filename       string  `json:"filename"`
	FileSize       int64   `json:"file_size"`
	EstimatedTimeS int64   `json:"estimated_time_s"`
	FilamentWeight float32 `json:"filament_weight_g"`
	FilamentType   string  `json:"filament_type"`
	ThumbnailURL   string  `json:"thumbnail_url"`
	LayerHeight    float32 `json:"layer_height"`
	Infill         string  `json:"infill"`
	NozzleTemp     float32 `json:"nozzle_temp"`
	BedTemp        float32 `json:"bed_temp"`
	HasMultiColor  bool    `json:"has_multi_color"`
	ColorCount     int     `json:"color_count"`
}

// AnalyzeFile analyzes a 3MF or GCODE file
func AnalyzeFile(filePath, uploadDir string) (*AnalysisResult, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".3mf":
		return analyze3MF(filePath, uploadDir)
	case ".gcode", ".g":
		return analyzeGCODE(filePath, uploadDir)
	case ".gcode.3mf":
		return analyze3MF(filePath, uploadDir)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

// analyze3MF parses a 3MF file (ZIP archive)
func analyze3MF(filePath, uploadDir string) (*AnalysisResult, error) {
	result := &AnalysisResult{
		Filename: filepath.Base(filePath),
	}

	// Get file size
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	result.FileSize = info.Size()

	// Open ZIP archive
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open 3MF: %w", err)
	}
	defer r.Close()

	// Look for metadata and thumbnail
	for _, f := range r.File {
		// Extract thumbnail
		if strings.Contains(f.Name, "Metadata/plate_") && strings.HasSuffix(f.Name, ".png") {
			thumbnailPath := filepath.Join(uploadDir, "thumbnails", f.Name)
			if err := extractFile(f, thumbnailPath); err == nil {
				result.ThumbnailURL = "/uploads/thumbnails/" + f.Name
			}
		}

		// Parse metadata XML
		if strings.HasSuffix(f.Name, "metadata.xml") || strings.HasSuffix(f.Name, "model.metadata.3mf") {
			if err := parseMetadata(f, result); err != nil {
				// Log or handle error if needed, but don't just leave it empty if possible
				continue
			}
		}

		// Check for multi-color (multiple materials)
		if strings.HasSuffix(f.Name, "model.metadata.3mf") {
			if colors, err := countColors(f); err == nil {
				result.ColorCount = colors
				result.HasMultiColor = colors > 1
			}
		}
	}

	return result, nil
}

// parseMetadata extracts slicing information from metadata XML
func parseMetadata(file *zip.File, result *AnalysisResult) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// Read and parse XML
	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	var metadata ThreeMFMetadata
	if err := xml.Unmarshal(data, &metadata); err != nil {
		return err
	}

	// Extract first plate info
	if len(metadata.Slicing.Plate) > 0 {
		plate := metadata.Slicing.Plate[0]
		result.EstimatedTimeS = plate.PrintTime
		result.FilamentWeight = plate.FilamentWeight
		result.LayerHeight = plate.LayerHeight
		result.Infill = plate.Infill
		result.NozzleTemp = plate.NozzleTemper
		result.BedTemp = plate.BedTemper
	}

	// Extract material info
	if len(metadata.Materials) > 0 {
		result.FilamentType = metadata.Materials[0].Type
	}

	return nil
}

// countColors counts the number of unique materials/colors
func countColors(file *zip.File) (int, error) {
	rc, err := file.Open()
	if err != nil {
		return 0, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return 0, err
	}

	var metadata ThreeMFMetadata
	if err := xml.Unmarshal(data, &metadata); err != nil {
		return 0, err
	}

	return len(metadata.Materials), nil
}

// analyzeGCODE parses a GCODE file for metadata
func analyzeGCODE(filePath, uploadDir string) (*AnalysisResult, error) {
	result := &AnalysisResult{
		Filename: filepath.Base(filePath),
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	result.FileSize = info.Size()

	// Open and read header comments
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := make([]byte, 8192) // Read first 8KB for metadata
	n, _ := file.Read(buf)
	header := string(buf[:n])

	// Parse common slicer metadata
	lines := strings.Split(header, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "; estimated printing time") {
			result.EstimatedTimeS = parseTimeEstimate(line)
		}
		if strings.HasPrefix(line, "; filament used") {
			result.FilamentWeight = parseFilamentUsed(line)
		}
		if strings.HasPrefix(line, "; filament_type =") {
			result.FilamentType = strings.TrimSpace(strings.TrimPrefix(line, "; filament_type ="))
		}
	}

	return result, nil
}

// extractFile extracts a file from ZIP to destination
func extractFile(file *zip.File, destPath string) error {
	// Create directory if needed
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

// parseTimeEstimate extracts time from GCODE comment
func parseTimeEstimate(line string) int64 {
	// Format: "; estimated printing time = 2h 15m 30s"
	parts := strings.Split(line, "=")
	if len(parts) < 2 {
		return 0
	}

	timeStr := strings.TrimSpace(parts[1])
	var total int64

	// Parse hours, minutes, seconds
	fmt.Sscanf(timeStr, "%dh %dm %ds", &total)

	// Simple parsing - can be enhanced
	if strings.Contains(timeStr, "h") {
		var h, m, s int
		fmt.Sscanf(timeStr, "%dh %dm %ds", &h, &m, &s)
		total = int64(h*3600 + m*60 + s)
	} else if strings.Contains(timeStr, "m") {
		var m, s int
		fmt.Sscanf(timeStr, "%dm %ds", &m, &s)
		total = int64(m*60 + s)
	}

	return total
}

// parseFilamentUsed extracts filament weight from GCODE comment
func parseFilamentUsed(line string) float32 {
	// Format: "; filament used [mm] = 1234.5, [g] = 34.56"
	parts := strings.Split(line, "[g]")
	if len(parts) < 2 {
		return 0
	}

	// Extract gram value
	valueStr := strings.TrimSpace(strings.TrimPrefix(parts[1], "="))
	var weight float32
	fmt.Sscanf(valueStr, "%f", &weight)
	return weight
}

// SaveAnalysisResult saves analysis to JSON file
func SaveAnalysisResult(result *AnalysisResult, outputDir string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	outputPath := filepath.Join(outputDir, result.Filename+".analysis.json")
	return os.WriteFile(outputPath, data, 0644)
}
