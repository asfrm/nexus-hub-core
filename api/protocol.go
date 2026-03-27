package api

import (
	"time"
)

type CpuCore struct {
	ID           int     `json:"id"`
	FrequencyMHz uint64  `json:"frequency_mhz"`
	UsagePercent float32 `json:"usage_percent"`
}

type HostHardwareState struct {
	CPUModel       string    `json:"cpu_model"`
	CPUUsageTotal  float32   `json:"cpu_usage_total"`
	CPUPackageTemp float32   `json:"cpu_package_temp"`
	CPUCores       []CpuCore `json:"cpu_cores"`
	GPUName        string    `json:"gpu_name"`
	GPULoadPercent float32   `json:"gpu_load_percent"`
	GPUMemUsedMB   uint64    `json:"gpu_mem_used_mb"`
	GPUMemTotalMB  uint64    `json:"gpu_mem_total_mb"`
	GPUTempCelsius float32   `json:"gpu_temp_celsius"`
	GPUPowerWatts  float32   `json:"gpu_power_watts"`
	GPUFanSpeedRPM uint32    `json:"gpu_fan_speed_rpm"`
	RAMUsedMB      uint64    `json:"ram_used_mb"`
	RAMTotalMB     uint64    `json:"ram_total_mb"`
	Uptime         string    `json:"uptime_formatted"`
}

// NexusPacket в Go реализуется через общую структуру, 
// так как в Go нет Enum с данными (как в Rust)
type NexusPacket struct {
	EventType string             `json:"event_type"`
	Payload   interface{}        `json:"payload"`
	Log       *LogPayload        `json:"log,omitempty"`
}

type LogPayload struct {
	Module    string    `json:"module"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	Timestamp time.Time `json:"timestamp"`
}