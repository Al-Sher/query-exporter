package metric_cache

import (
	"database/sql"
	"fmt"
	scanqueries "github.com/Al-Sher/query-exporter/internal/scan-queries"
	"go.uber.org/zap"
	"strings"
	"sync"
)

// MetricCache структура кэша метрик
type MetricCache struct {
	sync.RWMutex
	metrics       map[string]map[string]float64
	labels        map[string]map[string]map[string]string
	labelNames    map[string][]string
	changedLabels map[string]bool
}

// NewMetricCache создаем кэш метрик
func NewMetricCache() *MetricCache {
	return &MetricCache{
		metrics:       make(map[string]map[string]float64),
		labels:        make(map[string]map[string]map[string]string),
		labelNames:    make(map[string][]string),
		changedLabels: make(map[string]bool),
	}
}

// Store сохранить метрику в кэш
func (mc *MetricCache) Store(metricName string, labelValues map[string]string, value float64) {
	labelHash := hashLabelValues(labelValues)

	expectedLabels, ok := mc.labelNames[metricName]
	if !ok || len(expectedLabels) != len(labelValues) {
		mc.metrics[metricName] = make(map[string]float64)
		mc.labels[metricName] = make(map[string]map[string]string)
		mc.labelNames[metricName] = make([]string, len(labelValues))
		mc.changedLabels[metricName] = true
	}

	mc.metrics[metricName][labelHash] = value
	mc.labels[metricName][labelHash] = labelValues
}

// Get получить значения метрики
func (mc *MetricCache) Get(metricName string) (map[string]float64, map[string]map[string]string, bool) {
	values := make(map[string]float64)
	labels := make(map[string]map[string]string)

	for k, v := range mc.metrics[metricName] {
		values[k] = v
	}

	for k, v := range mc.labels[metricName] {
		labels[k] = v
	}

	changedLabels := mc.changedLabels[metricName]
	mc.changedLabels[metricName] = false

	return values, labels, changedLabels
}

// UpdateMetrics обновить метрики
func (mc *MetricCache) UpdateMetrics(logger *zap.Logger, q scanqueries.Query, rows *sql.Rows) {
	mc.RLock()
	defer mc.RUnlock()

	columns, err := rows.Columns()
	if err != nil {
		logger.Error(fmt.Sprintf("не вышло получить колонки метрики %s: %v", q.Name, err))
		return
	}

	var metricField string

	for _, col := range columns {
		lowerCol := strings.ToLower(col)
		if lowerCol == "count" || lowerCol == "quantity" {
			metricField = col
		}
	}

	if metricField == "" {
		logger.Error(fmt.Sprintf("не обнаружено значение метрики в запросе. Поддерживаются только count, quantity"))
		return
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
			logger.Error(fmt.Sprintf("не вышло получить значения метрики %s: %v", q.Name, err))
		}

		for i, col := range columns {
			if col == metricField {
				value = convertToFloat(values[i])
				continue
			}

			labels[col] = fmt.Sprintf("%v", values[i])
		}
		mc.Store(q.Name, labels, value)
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
