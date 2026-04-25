// Package parser provides analysis for slicer-processed files (GCODE and Sliced 3MF)
package parser

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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

var modelMap = map[string]string{
	"N1":  "Bambu Lab A1 mini",
	"N2S": "Bambu Lab A1",
	"X1":  "Bambu Lab X1",
	"X1C": "Bambu Lab X1C",
	"P1P": "Bambu Lab P1P",
	"P1S": "Bambu Lab P1S",
}

// AnalyzeFile analyzes GCODE and Sliced 3MF files
func AnalyzeFile(filePath, uploadDir string) (*AnalysisResult, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Handle special .gcode.3mf case with fallback
	if strings.HasSuffix(strings.ToLower(filePath), ".gcode.3mf") {
		result, err := analyze3MF(filePath, uploadDir)
		if err == nil {
			return result, nil
		}
		return analyzeGCODE(filePath, uploadDir)
	}

	switch ext {
	case ".3mf":
		return analyze3MF(filePath, uploadDir)
	case ".gcode", ".g":
		return analyzeGCODE(filePath, uploadDir)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

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
		return nil, fmt.Errorf("failed to open 3MF as zip: %w", err)
	}
	defer r.Close()

	hasSlicingConfig := false
	hasGCode := false

	for _, f := range r.File {
		if strings.Contains(f.Name, "Metadata/") || strings.HasSuffix(f.Name, ".config") || strings.HasSuffix(f.Name, ".xml") {
			hasSlicingConfig = true
		}
		if strings.HasSuffix(f.Name, ".gcode") {
			hasGCode = true
			_ = parseGCODEFromZip(f, result)
		}
		if (strings.Contains(f.Name, "Metadata/plate_") || strings.Contains(f.Name, ".png") || strings.Contains(f.Name, ".jpg")) &&
			(strings.HasSuffix(f.Name, ".png") || strings.HasSuffix(f.Name, ".jpg")) {
			safeName := strings.ReplaceAll(f.Name, "/", "_")
			thumbnailPath := filepath.Join(uploadDir, "thumbnails", safeName)
			if err := extractFile(f, thumbnailPath); err == nil {
				result.ThumbnailURL = "/uploads/thumbnails/" + safeName
			}
		}
		if strings.HasPrefix(f.Name, "Metadata/plate_") && strings.HasSuffix(f.Name, ".json") {
			_ = parsePlateJSON(f, result)
		}
		if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".3mf") || strings.HasSuffix(f.Name, ".config") {
			_ = parseMetadataXML(f, result)
		}
	}

	if result.PrinterModel == "Unknown" || result.FilamentType == "Unknown" {
		guessFromFilename(result)
	}

	if !hasSlicingConfig && !hasGCode {
		return nil, fmt.Errorf("file is a raw 3D model, not a sliced printer file")
	}

	result.IsSliced = true
	return result, nil
}

func parsePlateJSON(file *zip.File, result *AnalysisResult) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	byteValue, _ := io.ReadAll(rc)
	var data map[string]interface{}
	if err := json.Unmarshal(byteValue, &data); err != nil {
		return err
	}

	if timeStr, ok := data["estimated_time"].(string); ok && result.EstimatedTimeS == 0 {
		result.EstimatedTimeS = parseTimeEstimate(timeStr)
	} else if timeFloat, ok := data["estimated_time_s"].(float64); ok && result.EstimatedTimeS == 0 {
		result.EstimatedTimeS = int64(timeFloat)
	}
	if filaments, ok := data["filament_type"].([]interface{}); ok && len(filaments) > 0 && result.FilamentType == "Unknown" {
		result.FilamentType = fmt.Sprintf("%v", filaments[0])
	}
	if weights, ok := data["filament_weights"].([]interface{}); ok && len(weights) > 0 && result.FilamentWeight == 0 {
		if w, ok := weights[0].(float64); ok {
			result.FilamentWeight = float32(w)
		}
	}
	return nil
}

func parseGCODEFromZip(file *zip.File, result *AnalysisResult) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	scanner := bufio.NewScanner(rc)
	linesRead := 0
	for scanner.Scan() && linesRead < 2000 {
		linesRead++
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, ";") {
			continue
		}
		if strings.Contains(line, "=") || strings.Contains(line, ":") {
			var key, val string
			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				key = strings.TrimSpace(strings.TrimPrefix(parts[0], ";"))
				val = strings.Trim(strings.TrimSpace(parts[1]), "; \"'")
			} else {
				parts := strings.SplitN(line, ":", 2)
				key = strings.TrimSpace(strings.TrimPrefix(parts[0], ";"))
				val = strings.Trim(strings.TrimSpace(parts[1]), "; \"'")
			}
			if key != "" && val != "" {
				if strings.Contains(key, "estimated printing time") && result.EstimatedTimeS == 0 {
					result.EstimatedTimeS = parseTimeEstimate(val)
				} else if strings.Contains(key, "filament_type") && result.FilamentType == "Unknown" {
					types := strings.Split(val, ",")
					result.FilamentType = types[0]
				} else if (strings.Contains(key, "printer_model") || strings.Contains(key, "model_id")) && result.PrinterModel == "Unknown" {
					result.PrinterModel = val
				} else if key == "bed_temperature" || key == "first_layer_bed_temperature" {
					if v, err := strconv.ParseFloat(val, 32); err == nil && result.BedTemp == 0 {
						result.BedTemp = float32(v)
					}
				} else if key == "temperature" || key == "first_layer_temperature" {
					if v, err := strconv.ParseFloat(val, 32); err == nil && result.NozzleTemp == 0 {
						result.NozzleTemp = float32(v)
					}
				}
			}
		}
	}
	return nil
}

func parseMetadataXML(file *zip.File, result *AnalysisResult) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	decoder := xml.NewDecoder(rc)
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			var key, value string
			for _, attr := range se.Attr {
				if attr.Name.Local == "key" || attr.Name.Local == "name" {
					key = attr.Value
				}
				if attr.Name.Local == "value" || attr.Name.Local == "content" {
					value = attr.Value
				}
				// Handle direct attributes in <filament ...> tag
				if se.Name.Local == "filament" {
					if attr.Name.Local == "type" && result.FilamentType == "Unknown" {
						result.FilamentType = attr.Value
					}
					if (attr.Name.Local == "used_g" || attr.Name.Local == "weight") && result.FilamentWeight == 0 {
						if v, err := strconv.ParseFloat(attr.Value, 32); err == nil {
							result.FilamentWeight = float32(v)
						}
					}
				}
				// Handle attributes in <metadata ...> or other tags
				if attr.Name.Local == "printer_model" || attr.Name.Local == "printer_model_id" {
					if result.PrinterModel == "Unknown" {
						result.PrinterModel = attr.Value
					}
				}
				if attr.Name.Local == "print_time" || attr.Name.Local == "prediction" {
					if v, err := strconv.ParseInt(attr.Value, 10, 64); err == nil && result.EstimatedTimeS == 0 {
						result.EstimatedTimeS = v
					}
				}
			}
			if key != "" && value != "" {
				applyMetadataValue(key, value, result)
			}
		}
	}
	// Map model ID to name
	if readable, ok := modelMap[result.PrinterModel]; ok {
		result.PrinterModel = readable
	}
	return nil
}

func applyMetadataValue(key, value string, result *AnalysisResult) {
	key = strings.ToLower(key)
	switch key {
	case "printer_model", "printer_model_id", "model_id":
		if result.PrinterModel == "Unknown" {
			result.PrinterModel = value
		}
	case "filament_type", "filament_id":
		if result.FilamentType == "Unknown" {
			result.FilamentType = value
		}
	case "estimated_time", "print_time", "prediction":
		if v, err := strconv.ParseInt(value, 10, 64); err == nil && result.EstimatedTimeS == 0 {
			result.EstimatedTimeS = v
		}
	case "weight":
		if v, err := strconv.ParseFloat(value, 32); err == nil && result.FilamentWeight == 0 {
			result.FilamentWeight = float32(v)
		}
	case "layer_height":
		if v, err := strconv.ParseFloat(value, 32); err == nil && result.LayerHeight == 0 {
			result.LayerHeight = float32(v)
		}
	}
}

func parseConfigINI(file *zip.File, result *AnalysisResult) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), "; \"'")
			if strings.Contains(key, "filament_type") && result.FilamentType == "Unknown" {
				result.FilamentType = val
			}
			if (strings.Contains(key, "printer_model") || strings.Contains(key, "model_id")) && result.PrinterModel == "Unknown" {
				result.PrinterModel = val
			}
		}
	}
	return nil
}

func guessFromFilename(result *AnalysisResult) {
	upperName := strings.ToUpper(result.Filename)
	filaments := []string{"PETG", "PLA", "ABS", "ASA", "TPU", "PC", "PET", "PA", "CF"}
	for _, f := range filaments {
		if strings.Contains(upperName, f) {
			result.FilamentType = f
			break
		}
	}
	if strings.Contains(upperName, "X1C") || strings.Contains(upperName, "X1") {
		result.PrinterModel = "Bambu Lab X1C"
	} else if strings.Contains(upperName, "P1P") {
		result.PrinterModel = "Bambu Lab P1P"
	} else if strings.Contains(upperName, "P1S") {
		result.PrinterModel = "Bambu Lab P1S"
	} else if strings.Contains(upperName, "A1") {
		result.PrinterModel = "Bambu Lab A1"
	}
}

func analyzeGCODE(filePath, uploadDir string) (*AnalysisResult, error) {
	result := &AnalysisResult{
		Filename:     filepath.Base(filePath),
		FilamentType: "Unknown",
		PrinterModel: "Generic",
		IsSliced:     true,
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	linesRead := 0
	for scanner.Scan() && linesRead < 10000 {
		linesRead++
		line := scanner.Text()
		if !strings.HasPrefix(line, ";") {
			continue
		}
		line = strings.TrimSpace(line)
		if strings.Contains(line, "=") || strings.Contains(line, ":") {
			var key, val string
			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				key = strings.TrimSpace(strings.TrimPrefix(parts[0], ";"))
				val = strings.Trim(strings.TrimSpace(parts[1]), "; \"'")
			} else {
				parts := strings.SplitN(line, ":", 2)
				key = strings.TrimSpace(strings.TrimPrefix(parts[0], ";"))
				val = strings.Trim(strings.TrimSpace(parts[1]), "; \"'")
			}
			if key != "" && val != "" {
				if strings.Contains(key, "estimated printing time") {
					result.EstimatedTimeS = parseTimeEstimate(val)
				} else if strings.Contains(key, "filament used") && strings.Contains(line, "[g]") {
					result.FilamentWeight = parseFilamentUsed(line)
				} else if strings.Contains(key, "filament_type") {
					result.FilamentType = val
				} else if strings.Contains(key, "printer_model") || strings.Contains(key, "model_id") {
					result.PrinterModel = val
				} else if strings.Contains(key, "nozzle_temperature") || (key == "temperature") {
					if v, err := strconv.ParseFloat(val, 32); err == nil {
						result.NozzleTemp = float32(v)
					}
				} else if strings.Contains(key, "bed_temperature") {
					if v, err := strconv.ParseFloat(val, 32); err == nil {
						result.BedTemp = float32(v)
					}
				} else if key == "layer_height" {
					if v, err := strconv.ParseFloat(val, 32); err == nil {
						result.LayerHeight = float32(v)
					}
				}
			}
		}
	}
	if result.PrinterModel == "Generic" || result.FilamentType == "Unknown" {
		guessFromFilename(result)
	}
	return result, nil
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

func parseTimeEstimate(timeStr string) int64 {
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
	valPart := strings.Split(parts[1], "=")
	if len(valPart) < 2 {
		return 0
	}
	valueStr := strings.Trim(strings.TrimSpace(valPart[1]), "; \"'")
	var weight float32
	fmt.Sscanf(valueStr, "%f", &weight)
	return weight
}
