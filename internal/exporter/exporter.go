package exporter

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Al-Sher/query-exporter/internal/config"
	"github.com/Al-Sher/query-exporter/internal/db"
	metriccache "github.com/Al-Sher/query-exporter/internal/metric-cache"
	scanqueries "github.com/Al-Sher/query-exporter/internal/scan-queries"
	"github.com/Al-Sher/query-exporter/internal/validator"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"sync"
	"time"
)

// Exporter структура приложения
type Exporter struct {
	Queries     map[string]scanqueries.Query
	DB          *sql.DB
	Metrics     map[string]prometheus.Collector
	server      *http.Server
	wg          sync.WaitGroup
	reloadMutex sync.Mutex
	qv          *validator.QueryValidator
	watcher     *scanqueries.QueryWatcher
	stopChan    chan struct{}
	config      *config.Config
	cache       *metriccache.MetricCache
}

// NewExporter создать новый экземпляр Exporter
func NewExporter(c config.Config) *Exporter {
	qe := &Exporter{
		Queries:  make(map[string]scanqueries.Query),
		Metrics:  make(map[string]prometheus.Collector),
		qv:       validator.NewQueryValidator("query_exporter"),
		config:   &c,
		stopChan: make(chan struct{}),
	}

	dbConnection, dbMetrics, err := db.NewConnection(c)
	if err != nil {
		panic(fmt.Errorf("не вышло подключиться к БД: %w", err))
	}
	qe.DB = dbConnection

	watcher, err := scanqueries.NewWatcher(c.QueriesPath, 2*time.Second)
	if err != nil {
		panic(fmt.Errorf("не вышло запустить наблюдателя за изменением файлов: %w", err))
	}

	qe.watcher = watcher

	q, err := scanqueries.Scan(c.QueriesPath, *qe.qv)
	if err != nil {
		panic(fmt.Errorf("не вышло получить запросы: %w", err))
	}

	qe.Queries = q

	cache := metriccache.NewMetricCache()
	qe.cache = cache

	qe.wg.Add(1)
	go qe.runUpdater()

	qe.wg.Add(1)
	go qe.runWatcher()

	prometheus.MustRegister(qe.qv, dbMetrics, qe)

	return qe
}

// runWatcher запуск наблюдателя
func (qe *Exporter) runWatcher() {
	defer qe.wg.Done()
	go qe.watcher.Run()

	for {
		select {
		case <-qe.watcher.EventChan:
			time.Sleep(1 * time.Second)
			if err := qe.ReloadQueries(); err != nil {
				log.Printf("не вышло обвить запросы: %v", err)
			}
		case <-qe.stopChan:
			qe.watcher.Close()
			return
		}
	}
}

// runUpdater запуск обновления метрик
func (qe *Exporter) runUpdater() {
	defer qe.wg.Done()

	ticker := time.NewTicker(time.Duration(qe.config.Interval) * time.Second)
	defer ticker.Stop()

	qe.updateMetrics()

	for {
		select {
		case <-ticker.C:
			qe.updateMetrics()
		case <-qe.stopChan:
			return
		}
	}
}

// ReloadQueries обновление запросов метрик
func (qe *Exporter) ReloadQueries() error {
	qe.reloadMutex.Lock()
	defer qe.reloadMutex.Unlock()

	newQueries, err := scanqueries.Scan(qe.config.QueriesPath, *qe.qv)
	if err != nil {
		return err
	}

	qe.Queries = newQueries
	qe.clearDisabledMetrics()

	return nil
}

// clearDisabledMetrics очистка удаленных метрик
func (qe *Exporter) clearDisabledMetrics() {
	for name := range qe.Metrics {
		if _, exists := qe.Queries[name]; !exists {
			delete(qe.Metrics, name)
		}
	}
}

// StartServer запуск http-сервера
func (qe *Exporter) StartServer() {
	router := http.NewServeMux()
	router.Handle(qe.config.MetricsPath, promhttp.Handler())

	qe.server = &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: router,
	}

	go func() {
		log.Println("Starting server")
		if err := qe.server.ListenAndServe(); err != nil {
			log.Printf("не вышло запустить веб-сервер: %w", err)
		}
	}()
}

// Shutdown мягкое завершение сборщика метрик
func (qe *Exporter) Shutdown(ctx context.Context) error {
	close(qe.stopChan)
	qe.wg.Wait()

	if qe.server != nil {
		if err := qe.server.Shutdown(ctx); err != nil {
			return err
		}
	}

	if qe.DB != nil {
		if err := qe.DB.Close(); err != nil {
			return err
		}
	}

	return nil
}

// updateMetrics обновление метрик
func (qe *Exporter) updateMetrics() {
	var wg sync.WaitGroup
	sem := make(chan struct{}, qe.config.PoolSize)

	for metricName, q := range qe.Queries {
		wg.Add(1)
		sem <- struct{}{}

		go func(name string, q scanqueries.Query) {
			defer wg.Done()
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(qe.config.Timeout)*time.Second)
			defer cancel()

			conn, err := qe.DB.Conn(ctx)
			if err != nil {
				log.Printf("не вышло подключиться к базе для обновления метрики %s: %v", name, err)
				return
			}

			defer func(conn *sql.Conn) {
				err := conn.Close()
				if err != nil {
					log.Printf("не вышло закрыть соединение к БД: %v", err)
				}
			}(conn)

			rows, err := conn.QueryContext(ctx, q.Sql)
			if err != nil {
				log.Printf("не вышло выполнить запрос %s: %v", q.Sql, err)
				return
			}

			metriccache.UpdateMetrics(qe.cache, q, rows)
		}(metricName, q)
	}
}

// Describe добавление дескриптора к метрикам
func (qe *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range qe.Metrics {
		metric.Describe(ch)
	}
}

// Collect сборка метрик
func (qe *Exporter) Collect(ch chan<- prometheus.Metric) {
	for metricName := range qe.Queries {
		values, labels := qe.cache.Get(metricName)

		qe.collectGauge(metricName, values, labels, ch)
	}
}

// collectGauge сборка метрик с типом gauge
func (qe *Exporter) collectGauge(metricName string, values map[string]float64, labels map[string]map[string]string, ch chan<- prometheus.Metric) {
	var labelNames []string
	var labelValues = make(map[string][]string)
	var isNames = false

	for hash, label := range labels {
		for k, v := range label {
			if !isNames {
				labelNames = append(labelNames, k)
			}
			labelValues[hash] = append(labelValues[hash], v)
		}

		isNames = true
	}

	if _, ok := qe.Metrics[metricName]; !ok {
		qe.Metrics[metricName] = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: metricName,
				Help: metricName,
			},
			labelNames,
		)
	}

	if gauge, ok := qe.Metrics[metricName].(*prometheus.GaugeVec); ok {
		for labelHash, value := range values {
			gauge.WithLabelValues(labelValues[labelHash]...).Set(value)
		}
		gauge.Collect(ch)
	}
}
