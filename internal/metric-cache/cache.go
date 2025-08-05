package metric_cache

import (
	"database/sql"
	"fmt"
	scanqueries "github.com/Al-Sher/query-exporter/internal/scan-queries"
	"log"
	"strings"
	"sync"
)

// MetricCache структура кэша метрик
type MetricCache struct {
	sync.RWMutex
	metrics map[string]map[string]float64
	labels  map[string]map[string]map[string]string
}

// NewMetricCache создаем кэш метрик
func NewMetricCache() *MetricCache {
	return &MetricCache{
		metrics: make(map[string]map[string]float64),
		labels:  make(map[string]map[string]map[string]string),
	}
}

// Store сохранить метрику в кэш
func (mc *MetricCache) Store(metricName string, labelValues map[string]string, value float64) {
	mc.Lock()
	defer mc.Unlock()

	labelHash := hashLabelValues(labelValues)

	if _, ok := mc.metrics[metricName]; !ok {
		mc.metrics[metricName] = make(map[string]float64)
		mc.labels[metricName] = make(map[string]map[string]string)
	}

	mc.metrics[metricName][labelHash] = value
	mc.labels[metricName][labelHash] = labelValues
}

// Get получить значения метрики
func (mc *MetricCache) Get(metricName string) (map[string]float64, map[string]map[string]string) {
	mc.RLock()
	defer mc.RUnlock()

	values := make(map[string]float64)
	labels := make(map[string]map[string]string)

	for k, v := range mc.metrics[metricName] {
		values[k] = v
	}

	for k, v := range mc.labels[metricName] {
		labels[k] = v
	}

	return values, labels
}

// UpdateMetrics обновить метрики
func UpdateMetrics(m *MetricCache, q scanqueries.Query, rows *sql.Rows) {
	columns, err := rows.Columns()
	if err != nil {
		log.Fatalf("не вышло получить колонки метрики %s: %v", q.Name, err)
		return
	}

	var metricField string

	for _, col := range columns {
		lowerCol := strings.ToLower(col)
		if lowerCol == "count" || lowerCol == "quantity" {
			metricField = col
		}
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}
	var value float64

	for rows.Next() {
		var labels = make(map[string]string, len(columns))

		if err := rows.Scan(valuePtrs...); err != nil {
			log.Fatalf("не вышло получить значения метрики %s: %v", q.Name, err)
		}

		for i, col := range columns {
			if col == metricField {
				value = convertToFloat(values[i])
				continue
			}

			labels[col] = fmt.Sprintf("%v", values[i])
		}
		m.Store(q.Name, labels, value)
	}
}

// convertToFloat преобразование значения из БД в float
func convertToFloat(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int64:
		return float64(v)
	case uint64:
		return float64(v)
	case int32:
		return float64(v)
	case uint32:
		return float64(v)
	case int16:
		return float64(v)
	case uint16:
		return float64(v)
	case int8:
		return float64(v)
	case uint8:
		return float64(v)
	case int:
		return float64(v)
	default:
		return 0
	}
}

// hashLabelValues сборка хэша метрик
func hashLabelValues(values map[string]string) string {
	var result []string

	for _, v := range values {
		result = append(result, v)
	}

	return strings.Join(result, ":")
}
