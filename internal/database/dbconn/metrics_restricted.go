package dbconn

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// metricsRestrictedCollector implements the Prometheus collector interface.
// It reports all metrics returned by pgxpool.Pool.Stat().
// Adapted from github.com/dlmiddlecote/sqlstats
type metricsRestrictedCollector struct {
	pool *pgxpool.Pool

	// descriptions of exported metrics
	acquireCountDesc         *prometheus.Desc
	acquireDurationDesc      *prometheus.Desc
	acquiredConnsDesc        *prometheus.Desc
	canceledAcquireCountDesc *prometheus.Desc
	constructingConnsDesc    *prometheus.Desc
	emptyAcquireCountDesc    *prometheus.Desc
	idleConnsDesc            *prometheus.Desc
	maxConnsDesc             *prometheus.Desc
	totalConnsDesc           *prometheus.Desc
}

func newMetricsRestrictedCollector(pool *pgxpool.Pool, dbname, app string) *metricsRestrictedCollector {
	const (
		namespace = "src"
		subsystem = "pgxpool_conns"
	)

	labels := prometheus.Labels{
		"db_name":  dbname,
		"app_name": app,
	}

	return &metricsRestrictedCollector{
		pool: pool,
		acquireCountDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "acquire_count"),
			"The cumulative count of successful acquires from the pool.",
			nil,
			labels,
		),
		acquireDurationDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "acquire_duration"),
			"The total duration of all successful acquires from the pool.",
			nil,
			labels,
		),
		acquiredConnsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "acquired_conns"),
			"The number of currently acquired connections in the pool.",
			nil,
			labels,
		),
		canceledAcquireCountDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "canceled_acquires"),
			"The cumulative count of acquires from the pool that were canceled by a context.",
			nil,
			labels,
		),
		constructingConnsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "constructing_conns"),
			"The number of conns with construction in progress in the pool.",
			nil,
			labels,
		),
		emptyAcquireCountDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "empty_acquires"),
			"The cumulative count of successful acquires from the pool that waited for a resource to be released or constructed because the pool was empty.",
			nil,
			labels,
		),
		idleConnsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "idle_conns"),
			"The number of currently idle conns in the pool.",
			nil,
			labels,
		),
		maxConnsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "max_conns"),
			"The maximum size of the pool.",
			nil,
			labels,
		),
		totalConnsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "total_conns"),
			"The total number of resources currently in the pool.",
			nil,
			labels,
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (c metricsRestrictedCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquireCountDesc
	ch <- c.acquireDurationDesc
	ch <- c.acquiredConnsDesc
	ch <- c.canceledAcquireCountDesc
	ch <- c.constructingConnsDesc
	ch <- c.emptyAcquireCountDesc
	ch <- c.idleConnsDesc
	ch <- c.maxConnsDesc
	ch <- c.totalConnsDesc
}

// Collect implements the prometheus.Collector interface.
func (c metricsRestrictedCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.pool.Stat()

	ch <- prometheus.MustNewConstMetric(
		c.acquireCountDesc,
		prometheus.CounterValue,
		float64(stats.AcquireCount()),
	)
	ch <- prometheus.MustNewConstMetric(
		c.acquireDurationDesc,
		prometheus.CounterValue,
		float64(stats.AcquireDuration()),
	)
	ch <- prometheus.MustNewConstMetric(
		c.acquiredConnsDesc,
		prometheus.GaugeValue,
		float64(stats.AcquiredConns()),
	)
	ch <- prometheus.MustNewConstMetric(
		c.canceledAcquireCountDesc,
		prometheus.CounterValue,
		float64(stats.CanceledAcquireCount()),
	)
	ch <- prometheus.MustNewConstMetric(
		c.constructingConnsDesc,
		prometheus.GaugeValue,
		float64(stats.ConstructingConns()),
	)
	ch <- prometheus.MustNewConstMetric(
		c.emptyAcquireCountDesc,
		prometheus.CounterValue,
		float64(stats.EmptyAcquireCount()),
	)
	ch <- prometheus.MustNewConstMetric(
		c.idleConnsDesc,
		prometheus.GaugeValue,
		float64(stats.IdleConns()),
	)
	ch <- prometheus.MustNewConstMetric(
		c.maxConnsDesc,
		prometheus.GaugeValue,
		float64(stats.MaxConns()),
	)
	ch <- prometheus.MustNewConstMetric(
		c.totalConnsDesc,
		prometheus.GaugeValue,
		float64(stats.TotalConns()),
	)
}
