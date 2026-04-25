package slicer

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type PrintJobMeta struct {
	FilamentType   string
	EstimatedTimeS uint64
	PrinterModel   string
}

func ParseGCodeMetadata(filepath string) PrintJobMeta {
	file, err := os.Open(filepath)
	if err != nil {
		return PrintJobMeta{PrinterModel: "Unknown", FilamentType: "Unknown"}
	}
	defer file.Close()

	meta := PrintJobMeta{PrinterModel: "Unknown", FilamentType: "Unknown"}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// 1. Ищем модель принтера
		if strings.Contains(line, "printer_model") {
			if parts := strings.Split(line, "="); len(parts) == 2 {
				val := strings.Trim(strings.TrimSpace(parts[1]), "; ")
				// Проверяем, что это не переменная типа {printer_model}
				if val != "" && !strings.ContainsAny(val, "{}[]") {
					meta.PrinterModel = val
				}
			}
		}

		// 2. Ищем тип пластика
		if strings.Contains(line, "filament_type") {
			if parts := strings.Split(line, "="); len(parts) == 2 {
				val := strings.Trim(strings.TrimSpace(parts[1]), "; ")
				// Игнорируем шаблоны типа {input_filename_base}
				if val != "" && !strings.ContainsAny(val, "{}[]") {
					meta.FilamentType = val
				}
			}
		}

		// 3. Ищем время
		if strings.Contains(line, "estimated printing time") {
			if parts := strings.Split(line, "="); len(parts) == 2 {
				meta.EstimatedTimeS = parseTimeStr(strings.TrimSpace(parts[1]))
			}
		}
	}

	// Финальная проверка: если всё равно "Unknown",
	// но мы нашли что-то похожее на PETG/PLA в строке, можно добавить доп. логику
	if meta.PrinterModel == "Unknown" || meta.FilamentType == "Unknown" {
		// Fallback detection if needed
		meta.FilamentType = "PLA" // Default to PLA if unknown
	}
	return meta
}

func parseTimeStr(timeStr string) uint64 {
	var total uint64
	// Убираем лишние символы, оставляем только цифры и h/m/s
	cleanStr := strings.TrimPrefix(timeStr, "; ")
	parts := strings.Fields(cleanStr)
	for _, p := range parts {
		if strings.HasSuffix(p, "h") {
			v, _ := strconv.ParseUint(strings.TrimSuffix(p, "h"), 10, 64)
			total += v * 3600
		} else if strings.HasSuffix(p, "m") {
			v, _ := strconv.ParseUint(strings.TrimSuffix(p, "m"), 10, 64)
			total += v * 60
		} else if strings.HasSuffix(p, "s") {
			v, _ := strconv.ParseUint(strings.TrimSuffix(p, "s"), 10, 64)
			total += v
		}
	}
	return total
}
