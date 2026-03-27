package dispatcher

import (
	"nexus-hub-core/api"
	"nexus-hub-core/db" // Наш новый клиент
	"sync/atomic"
	"time"
)

// Глобальный переключатель (true = Авто, false = Ручной)
var AutoMode atomic.Bool

func StartDispatching(database *db.Database, tx chan api.NexusPacket) {
	ticker := time.NewTicker(5 * time.Second)

	for range ticker.C {
		// 1. Получаем все кластеры
		clusters, _ := database.GetClustersWithPrinters()

		for _, cluster := range clusters {
			// ЕСЛИ КЛАСТЕР ВЫКЛЮЧЕН — ПРОПУСКАЕМ
			if !cluster.IsEnabled {
				continue
			}

			// Ищем задачу для этого кластера (по модели)
			// (Здесь можно добавить метод GetNextJobForCluster)
			// Пока оставим логику ручного назначения, она приоритетнее.
		}
	}
}
