package db

import (
	"database/sql"
	"github.com/Al-Sher/query-exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
)

// PoolMetricsCollector метрики соединения с БД
type PoolMetricsCollector struct {
	db                    *sql.DB
	maxOpenConns          prometheus.Gauge
	openConns             prometheus.Gauge
	inUseConns            prometheus.Gauge
	idleConns             prometheus.Gauge
	waitCount             prometheus.Gauge
	waitDuration          prometheus.Gauge
	maxIdleClosed         prometheus.Gauge
	maxLifetimeClosed     prometheus.Gauge
	connectionErrors      prometheus.Counter
	connectionRetries     prometheus.Counter
	successfulConnections prometheus.Counter
}

// NewConnection создать соединение к БД и структуру метрик
func NewConnection(config config.Config) (db *sql.DB, metrics *PoolMetricsCollector, err error) {
	metrics = NewPoolMetricsCollector(nil, "query_exporter")

	db, err = sql.Open("postgres", config.Dsn)
	if err != nil {
		metrics.connectionErrors.Inc()
		return
	}

	if err = db.Ping(); err != nil {
		metrics.connectionErrors.Inc()
		return
	}

	db.SetMaxOpenConns(config.PoolSize)
	db.SetMaxIdleConns(config.PoolSize)
	metrics.db = db
	metrics.successfulConnections.Inc()

	return
}

// NewPoolMetricsCollector создать метрики БД
func NewPoolMetricsCollector(db *sql.DB, namespace string) *PoolMetricsCollector {
	return &PoolMetricsCollector{
		db: db,
		maxOpenConns: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_max_open_conns",
			Help:      "Maximum number of open connections to the database",
		}),
		openConns: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_open_conns",
			Help:      "Number of established connections both in use and idle",
		}),
		inUseConns: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_in_use_conns",
			Help:      "Number of connections currently in use",
		}),
		idleConns: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_idle_conns",
			Help:      "Number of idle connections",
		}),
		waitCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_wait_count",
			Help:      "Total number of connections waited for",
		}),
		waitDuration: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_wait_duration_seconds",
			Help:      "The total time blocked waiting for a new connection",
		}),
		maxIdleClosed: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_max_idle_closed",
			Help:      "Total number of connections closed due to SetMaxIdleConns",
		}),
		maxLifetimeClosed: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_max_lifetime_closed",
			Help:      "Total number of connections closed due to SetConnMaxLifetime",
		}),
		connectionErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "db_connection_errors_total",
			Help:      "Total number of connection errors",
		}),
		connectionRetries: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "db_connection_retries_total",
			Help:      "Total number of connection retries",
		}),
		successfulConnections: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "db_successful_connections_total",
			Help:      "Total number of successful connections",
		}),
	}
}

// Describe установка дескриптора метрик для избегания дублирования
func (c *PoolMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.maxOpenConns.Describe(ch)
	c.openConns.Describe(ch)
	c.inUseConns.Describe(ch)
	c.idleConns.Describe(ch)
	c.waitCount.Describe(ch)
	c.waitDuration.Describe(ch)
	c.maxIdleClosed.Describe(ch)
	c.maxLifetimeClosed.Describe(ch)
	c.connectionErrors.Describe(ch)
	c.connectionRetries.Describe(ch)
	c.successfulConnections.Describe(ch)
}

// Collect соборщик метрик
func (c *PoolMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()

	c.maxOpenConns.Set(float64(stats.MaxOpenConnections))
	c.openConns.Set(float64(stats.OpenConnections))
	c.inUseConns.Set(float64(stats.InUse))
	c.idleConns.Set(float64(stats.Idle))
	c.waitCount.Set(float64(stats.WaitCount))
	c.waitDuration.Set(stats.WaitDuration.Seconds())
	c.maxIdleClosed.Set(float64(stats.MaxIdleClosed))
	c.maxLifetimeClosed.Set(float64(stats.MaxLifetimeClosed))

	c.maxOpenConns.Collect(ch)
	c.openConns.Collect(ch)
	c.inUseConns.Collect(ch)
	c.idleConns.Collect(ch)
	c.waitCount.Collect(ch)
	c.waitDuration.Collect(ch)
	c.maxIdleClosed.Collect(ch)
	c.maxLifetimeClosed.Collect(ch)
	c.connectionErrors.Collect(ch)
	c.connectionRetries.Collect(ch)
	c.successfulConnections.Collect(ch)
}
