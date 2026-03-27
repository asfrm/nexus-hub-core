package monitor

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"nexus-hub-core/api"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/yusufpapurcu/wmi"
)

// Глобальные переменные для фонового обновления
var (
	bgCPUTemp float32
	bgGPUName = "Searching GPU..."
	bgGPULoad float32
	bgGPUMemU uint64
	bgGPUMemT uint64
	bgGPUTemp float32
)

type MSAcpi_ThermalZoneTemperature struct {
	CurrentTemperature uint32
}

// Фоновый сборщик (чтобы не тормозить запуск)
func startBackgroundWorkers() {
	go func() {
		for {
			// Опрос GPU
			out, err := exec.Command("nvidia-smi", "--query-gpu=name,utilization.gpu,memory.used,memory.total,temperature.gpu", "--format=csv,noheader,nounits").Output()
			if err == nil {
				parts := strings.Split(strings.TrimSpace(string(out)), ",")
				if len(parts) >= 5 {
					bgGPUName = strings.TrimSpace(parts[0])
					l, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 32)
					mu, _ := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
					mt, _ := strconv.ParseUint(strings.TrimSpace(parts[3]), 10, 64)
					t, _ := strconv.ParseFloat(strings.TrimSpace(parts[4]), 32)

					bgGPULoad, bgGPUMemU, bgGPUMemT, bgGPUTemp = float32(l), mu, mt, float32(t)
				}
			}

			// Опрос WMI (он виноват в долгих зависаниях)
			var dst []MSAcpi_ThermalZoneTemperature
			wmi.Query("SELECT CurrentTemperature FROM MSAcpi_ThermalZoneTemperature", &dst)
			if len(dst) > 0 {
				bgCPUTemp = float32(dst[0].CurrentTemperature)/10.0 - 273.15
			}

			time.Sleep(3 * time.Second)
		}
	}()
}

func StartLocalCollection(tx chan<- api.NexusPacket) {
	// Запускаем медленные датчики в фоне
	startBackgroundWorkers()

	// Узнаем имя проца (один раз)
	cpuInfo, _ := cpu.Info()
	cpuModel := "Unknown CPU"
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}

	for {
		// Опрос ядер (быстро)
		coresPercent, _ := cpu.Percent(time.Second, true)
		totalPercent, _ := cpu.Percent(0, false)
		cpuUsageTotal := float32(0)
		if len(totalPercent) > 0 {
			cpuUsageTotal = float32(totalPercent[0])
		}

		cores := make([]api.CpuCore, len(coresPercent))
		for i, p := range coresPercent {
			cores[i] = api.CpuCore{ID: i, FrequencyMHz: 0, UsagePercent: float32(p)}
		}

		vMem, _ := mem.VirtualMemory()
		up, _ := host.Uptime()

		metrics := api.HostHardwareState{
			CPUModel:       cpuModel,
			CPUUsageTotal:  cpuUsageTotal,
			CPUPackageTemp: bgCPUTemp,
			CPUCores:       cores,
			GPUName:        bgGPUName,
			GPULoadPercent: bgGPULoad,
			GPUMemUsedMB:   bgGPUMemU,
			GPUMemTotalMB:  bgGPUMemT,
			GPUTempCelsius: bgGPUTemp,
			RAMUsedMB:      vMem.Used / 1024 / 1024,
			RAMTotalMB:     vMem.Total / 1024 / 1024,
			Uptime:         fmt.Sprintf("%dh", up/3600),
		}

		// Отправляем на фронтенд
		tx <- api.NexusPacket{
			EventType: "HOST_METRICS",
			Payload:   metrics,
		}

		// ВОЗВРАЩАЕМ ЛОГИ В КОНСОЛЬ
		fmt.Printf("[%s] CPU: %5.1f%% (%2.0f°C) | GPU: %s %2.0f%% (%2.0f°C) | RAM: %dMB\n",
			time.Now().Format("15:04:05"), cpuUsageTotal, bgCPUTemp, bgGPUName, bgGPULoad, bgGPUTemp, metrics.RAMUsedMB)
	}
}
