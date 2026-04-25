package slicer

import (
	"nexus-hub-core/parser"
)

type PrintJobMeta struct {
	FilamentType   string
	EstimatedTimeS uint64
	PrinterModel   string
}

func ParseGCodeMetadata(filepath string) PrintJobMeta {
	// Call the new unified analyzer
	analysis, err := parser.AnalyzeFile(filepath, "./uploads")
	if err != nil {
		return PrintJobMeta{PrinterModel: "Unknown", FilamentType: "Unknown"}
	}

	return PrintJobMeta{
		FilamentType:   analysis.FilamentType,
		EstimatedTimeS: uint64(analysis.EstimatedTimeS),
		PrinterModel:   analysis.PrinterModel,
	}
}
