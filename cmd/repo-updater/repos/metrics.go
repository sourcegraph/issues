package repos

import (
	"context"
	"database/sql"

	"github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

const (
	tagFamily  = "family"
	tagID      = "id"
	tagState   = "state"
	tagSuccess = "success"
)

var (
	phabricatorUpdateTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "src_repoupdater_time_last_phabricator_sync",
		Help: "The last time a comprehensive Phabricator sync finished",
	}, []string{tagID})

	lastSync = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "src_repoupdater_syncer_sync_last_time",
		Help: "The last time a sync finished",
	}, []string{tagFamily})

	syncStarted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "src_repoupdater_syncer_start_sync",
		Help: "A sync was started",
	}, []string{tagFamily})

	syncedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "src_repoupdater_syncer_synced_repos_total",
		Help: "Total number of synced repositories",
	}, []string{tagState, tagFamily})

	syncErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "src_repoupdater_syncer_sync_errors_total",
		Help: "Total number of sync errors",
	}, []string{tagFamily})

	syncDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "src_repoupdater_syncer_sync_duration_seconds",
		Help: "Time spent syncing",
	}, []string{tagSuccess, tagFamily})

	purgeSuccess = promauto.NewCounter(prometheus.CounterOpts{
		Name: "src_repoupdater_purge_success",
		Help: "Incremented each time we remove a repository clone.",
	})

	purgeFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "src_repoupdater_purge_failed",
		Help: "Incremented each time we try and fail to remove a repository clone.",
	})

	schedError = promauto.NewCounter(prometheus.CounterOpts{
		Name: "src_repoupdater_sched_error",
		Help: "Incremented each time we encounter an error updating a repository.",
	})

	schedLoops = promauto.NewCounter(prometheus.CounterOpts{
		Name: "src_repoupdater_sched_loops",
		Help: "Incremented each time the scheduler loops.",
	})

	schedAutoFetch = promauto.NewCounter(prometheus.CounterOpts{
		Name: "src_repoupdater_sched_auto_fetch",
		Help: "Incremented each time the scheduler updates a managed repository due to hitting a deadline.",
	})

	schedManualFetch = promauto.NewCounter(prometheus.CounterOpts{
		Name: "src_repoupdater_sched_manual_fetch",
		Help: "Incremented each time the scheduler updates a repository due to user traffic.",
	})

	schedKnownRepos = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "src_repoupdater_sched_known_repos",
		Help: "The number of repositories that are managed by the scheduler.",
	})

	schedUpdateQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "src_repoupdater_sched_update_queue_length",
		Help: "The number of repositories that are currently queued for update",
	})
)

func MustRegisterMetrics(db dbutil.DB) {
	scanCount := func(sql string) (float64, error) {
		row := db.QueryRowContext(context.Background(), sql)
		var count int64
		err := row.Scan(&count)
		if err != nil {
			return 0, err
		}
		return float64(count), nil
	}

	scanNullFloat := func(q string) (sql.NullFloat64, error) {
		row := db.QueryRowContext(context.Background(), q)
		var v sql.NullFloat64
		err := row.Scan(&v)
		return v, err
	}

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "src_repoupdater_external_services_total",
		Help: "The total number of external services added",
	}, func() float64 {
		count, err := scanCount(`
-- source: cmd/repo-updater/repos/metrics.go:src_repoupdater_external_services_total
SELECT COUNT(*) FROM external_services
WHERE deleted_at IS NULL
`)
		if err != nil {
			log15.Error("Failed to get total external services", "err", err)
			return 0
		}
		return count
	})

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "src_repoupdater_user_external_services_total",
		Help: "The total number of external services added by users",
	}, func() float64 {
		count, err := scanCount(`
-- source: cmd/repo-updater/repos/metrics.go:src_repoupdater_user_external_services_total
SELECT COUNT(*) FROM external_services
WHERE namespace_user_id IS NOT NULL
AND deleted_at IS NULL
`)
		if err != nil {
			log15.Error("Failed to get total user external services", "err", err)
			return 0
		}
		return count
	})

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "src_repoupdater_user_repos_total",
		Help: "The total number of repositories added by users",
	}, func() float64 {
		count, err := scanCount(`
-- source: cmd/repo-updater/repos/metrics.go:src_repoupdater_user_repos_total
SELECT COUNT(*)
FROM
    repo r
WHERE
    EXISTS (
        SELECT
        FROM
            external_service_repos sr
            INNER JOIN external_services s ON s.id = sr.external_service_id
        WHERE
            s.namespace_user_id IS NOT NULL
            AND s.deleted_at IS NULL
            AND r.id = sr.repo_id
            AND r.deleted_at IS NULL)
`)
		if err != nil {
			log15.Error("Failed to get total user repositories", "err", err)
			return 0
		}
		return count
	})

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "src_repoupdater_user_with_external_services_total",
		Help: "The total number of users who have added external services",
	}, func() float64 {
		count, err := scanCount(`
-- source: cmd/repo-updater/repos/metrics.go:src_repoupdater_user_with_external_services_total
SELECT COUNT(DISTINCT(namespace_user_id)) AS total
FROM external_services
WHERE namespace_user_id IS NOT NULL
AND deleted_at IS NULL
`)
		if err != nil {
			log15.Error("Failed to get total users with external services", "err", err)
			return 0
		}
		return count
	})

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "src_repoupdater_queued_sync_jobs_total",
		Help: "The total number of queued sync jobs",
	}, func() float64 {
		count, err := scanCount(`
-- source: cmd/repo-updater/repos/metrics.go:src_repoupdater_queued_sync_jobs_total
SELECT COUNT(*) FROM external_service_sync_jobs WHERE state = 'queued'
`)
		if err != nil {
			log15.Error("Failed to get total queued sync jobs", "err", err)
			return 0
		}
		return count
	})

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "src_repoupdater_completed_sync_jobs_total",
		Help: "The total number of completed sync jobs",
	}, func() float64 {
		count, err := scanCount(`
-- source: cmd/repo-updater/repos/metrics.go:src_repoupdater_completed_sync_jobs_total
SELECT COUNT(*) FROM external_service_sync_jobs WHERE state = 'completed'
`)
		if err != nil {
			log15.Error("Failed to get total completed sync jobs", "err", err)
			return 0
		}
		return count
	})

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "src_repoupdater_errored_sync_jobs_total",
		Help: "The total number of errored sync jobs",
	}, func() float64 {
		count, err := scanCount(`
-- source: cmd/repo-updater/repos/metrics.go:src_repoupdater_errored_sync_jobs_total
SELECT COUNT(*) FROM external_service_sync_jobs WHERE state = 'errored'
`)
		if err != nil {
			log15.Error("Failed to get total errored sync jobs", "err", err)
			return 0
		}
		return count
	})

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "src_repoupdater_max_sync_backoff",
		Help: "The maximum number of seconds since any external service synced",
	}, func() float64 {
		seconds, err := scanNullFloat(`
-- source: cmd/repo-updater/repos/metrics.go:src_repoupdater_errored_sync_jobs_total
SELECT extract(epoch from max(now() - last_sync_at)) FROM external_services
WHERE deleted_at IS NULL
AND last_sync_at IS NOT NULL
`)
		if err != nil {
			log15.Error("Failed to get max sync backoff", "err", err)
			return 0
		}
		if !seconds.Valid {
			// This can happen when no external services have been synced and they all
			// have last_sync_at as null.
			return 0
		}
		return seconds.Float64
	})

}
