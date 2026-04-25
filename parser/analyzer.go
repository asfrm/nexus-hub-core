// Package parser provides analysis for slicer-processed files (GCODE and Sliced 3MF)
package parser

import (
	"archive/zip"
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ThreeMFMetadata represents 3MF slicing metadata (Bambu/Orca style)
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
	PrinterModel   string  `json:"printer_model"`
	IsSliced       bool    `json:"is_sliced"`
}

// AnalyzeFile analyzes GCODE and Sliced 3MF files
func AnalyzeFile(filePath, uploadDir string) (*AnalysisResult, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Handle special .gcode.3mf case
	if strings.HasSuffix(strings.ToLower(filePath), ".gcode.3mf") {
		return analyze3MF(filePath, uploadDir)
	}

	switch ext {
	case ".3mf":
		return analyze3MF(filePath, uploadDir)
	case ".gcode", ".g":
		return analyzeGCODE(filePath, uploadDir)
	default:
		return nil, fmt.Errorf("unsupported file type: %s. Only slicer-processed files (.gcode, .3mf) are supported", ext)
	}
}

// analyze3MF parses a 3MF file and verifies it is a SLICED file
func analyze3MF(filePath, uploadDir string) (*AnalysisResult, error) {
	result := &AnalysisResult{
		Filename:     filepath.Base(filePath),
		FilamentType: "Unknown",
		PrinterModel: "Unknown",
	}

	info, err := os.Stat(filePath)
	if err == nil {
		result.FileSize = info.Size()
	}

	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open 3MF: %w", err)
	}
	defer r.Close()

	hasSlicingConfig := false
	hasGCode := false

	for _, f := range r.File {
		// Sliced 3MFs (Orca/Bambu) MUST contain these or similar entries
		if strings.Contains(f.Name, "Metadata/slice_info") ||
			strings.Contains(f.Name, "Metadata/model.metadata") ||
			strings.Contains(f.Name, "metadata.xml") {
			hasSlicingConfig = true
		}

		// Check for presence of G-code within the archive (the real payload)
		if strings.HasSuffix(f.Name, ".gcode") {
			hasGCode = true
		}

		// Extract thumbnail
		if (strings.Contains(f.Name, "Metadata/plate_") || strings.Contains(f.Name, ".png")) &&
			(strings.HasSuffix(f.Name, ".png") || strings.HasSuffix(f.Name, ".jpg")) {
			safeName := strings.ReplaceAll(f.Name, "/", "_")
			thumbnailPath := filepath.Join(uploadDir, "thumbnails", safeName)
			if err := extractFile(f, thumbnailPath); err == nil {
				result.ThumbnailURL = "/uploads/thumbnails/" + safeName
			}
		}

		// Parse metadata XML (Priority to model.metadata.3mf as it contains more detail)
		if strings.HasSuffix(f.Name, "metadata.xml") || strings.HasSuffix(f.Name, "model.metadata.3mf") {
			_ = parseMetadata(f, result)
		}
	}

	// VALIDATION: If no slicing config or G-code found, it's a raw 3D model
	if !hasSlicingConfig && !hasGCode {
		return nil, fmt.Errorf("file is a raw 3D model, not a sliced printer file")
	}

	result.IsSliced = true
	return result, nil
}

// analyzeGCODE parses a GCODE file scanning for slicer-specific comments
func analyzeGCODE(filePath, uploadDir string) (*AnalysisResult, error) {
	result := &AnalysisResult{
		Filename:     filepath.Base(filePath),
		FilamentType: "PLA",
		PrinterModel: "Generic",
		IsSliced:     true,
	}

	info, err := os.Stat(filePath)
	if err == nil {
		result.FileSize = info.Size()
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	linesRead := 0
	for scanner.Scan() && linesRead < 5000 {
		linesRead++
		line := scanner.Text()
		if !strings.HasPrefix(line, ";") {
			continue
		}

		line = strings.TrimSpace(line)

		if strings.Contains(line, "estimated printing time") {
			result.EstimatedTimeS = parseTimeEstimate(line)
		}
		if strings.Contains(line, "filament used") && strings.Contains(line, "[g]") {
			result.FilamentWeight = parseFilamentUsed(line)
		}
		if strings.Contains(line, "filament_type") {
			result.FilamentType = extractValue(line)
		}
		if strings.Contains(line, "printer_model") || strings.Contains(line, "model_id") {
			result.PrinterModel = extractValue(line)
		}
		if strings.Contains(line, "nozzle_temperature") || strings.Contains(line, "temperature") {
			_, _ = fmt.Sscanf(extractValue(line), "%f", &result.NozzleTemp)
		}
		if strings.Contains(line, "bed_temperature") {
			_, _ = fmt.Sscanf(extractValue(line), "%f", &result.BedTemp)
		}
		if strings.Contains(line, "layer_height") {
			fmt.Sscanf(extractValue(line), "%f", &result.LayerHeight)
		}
	}

	return result, nil
}

func extractValue(line string) string {
	parts := strings.Split(line, "=")
	if len(parts) < 2 {
		parts = strings.Split(line, ":")
	}
	if len(parts) >= 2 {
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "; \"'")
		if val != "" && !strings.ContainsAny(val, "{}[]") {
			return val
		}
	}
	return ""
}

func parseMetadata(file *zip.File, result *AnalysisResult) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	var metadata ThreeMFMetadata
	if err := xml.Unmarshal(data, &metadata); err != nil {
		return err
	}

	if len(metadata.Slicing.Plate) > 0 {
		plate := metadata.Slicing.Plate[0]
		if plate.PrintTime > 0 {
			result.EstimatedTimeS = plate.PrintTime
		}
		if plate.FilamentWeight > 0 {
			result.FilamentWeight = plate.FilamentWeight
		}
		if plate.LayerHeight > 0 {
			result.LayerHeight = plate.LayerHeight
		}
		if plate.Infill != "" {
			result.Infill = plate.Infill
		}
		if plate.NozzleTemper > 0 {
			result.NozzleTemp = plate.NozzleTemper
		}
		if plate.BedTemper > 0 {
			result.BedTemp = plate.BedTemper
		}
	}

	if len(metadata.Materials) > 0 {
		// Use the material type if we don't have one yet
		if result.FilamentType == "Unknown" || result.FilamentType == "" {
			result.FilamentType = metadata.Materials[0].Type
		}
	}

	return nil
}

func extractFile(file *zip.File, destPath string) error {
	dir := filepath.Dir(destPath)
	_ = os.MkdirAll(dir, 0755)
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

func parseTimeEstimate(line string) int64 {
	parts := strings.Split(line, "=")
	if len(parts) < 2 {
		return 0
	}
	timeStr := strings.TrimSpace(parts[1])

	var total int64
	hParts := strings.Split(timeStr, "h")
	if len(hParts) > 1 {
		h, _ := strconv.ParseInt(strings.TrimSpace(hParts[0]), 10, 64)
		total += h * 3600
		timeStr = hParts[1]
	}
	mParts := strings.Split(timeStr, "m")
	if len(mParts) > 1 {
		m, _ := strconv.ParseInt(strings.TrimSpace(mParts[0]), 10, 64)
		total += m * 60
		timeStr = mParts[1]
	}
	sParts := strings.Split(timeStr, "s")
	s, _ := strconv.ParseInt(strings.TrimSpace(sParts[0]), 10, 64)
	total += s

	return total
}

func parseFilamentUsed(line string) float32 {
	parts := strings.Split(line, "[g]")
	if len(parts) < 2 {
		return 0
	}
	valueStr := strings.TrimSpace(strings.TrimPrefix(parts[1], "="))
	var weight float32
	fmt.Sscanf(valueStr, "%f", &weight)
	return weight
}
