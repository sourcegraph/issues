package background

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
	"github.com/keegancsmith/sqlf"

	cm "github.com/sourcegraph/sourcegraph/enterprise/internal/codemonitors"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codemonitors/email"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

const (
	eventRetentionInDays int = 7
)

func newTriggerQueryRunner(ctx context.Context, s *cm.Store, metrics codeMonitorsMetrics) *workerutil.Worker {
	options := workerutil.WorkerOptions{
		Name:        "code_monitors_trigger_jobs_worker",
		NumHandlers: 1,
		Interval:    5 * time.Second,
		Metrics:     metrics.workerMetrics,
	}
	worker := dbworker.NewWorker(ctx, createDBWorkerStoreForTriggerJobs(s), &queryRunner{s}, options)
	return worker
}

func newTriggerQueryEnqueuer(ctx context.Context, store *cm.Store) goroutine.BackgroundRoutine {
	enqueueActive := goroutine.NewHandlerWithErrorMessage(
		"code_monitors_trigger_query_enqueuer",
		func(ctx context.Context) error {
			return store.EnqueueTriggerQueries(ctx)
		})
	return goroutine.NewPeriodicGoroutine(ctx, 1*time.Minute, enqueueActive)
}

func newTriggerQueryResetter(ctx context.Context, s *cm.Store, metrics codeMonitorsMetrics) *dbworker.Resetter {
	workerStore := createDBWorkerStoreForTriggerJobs(s)

	options := dbworker.ResetterOptions{
		Name:     "code_monitors_trigger_jobs_worker_resetter",
		Interval: 1 * time.Minute,
		Metrics: dbworker.ResetterMetrics{
			Errors:              metrics.errors,
			RecordResetFailures: metrics.resetFailures,
			RecordResets:        metrics.resets,
		},
	}
	return dbworker.NewResetter(workerStore, options)
}

func newTriggerJobsLogDeleter(ctx context.Context, store *cm.Store) goroutine.BackgroundRoutine {
	deleteLogs := goroutine.NewHandlerWithErrorMessage(
		"code_monitors_trigger_jobs_log_deleter",
		func(ctx context.Context) error {
			// Delete logs without search results.
			err := store.DeleteObsoleteJobLogs(ctx)
			if err != nil {
				return err
			}
			// Delete old logs, even if they have search results.
			err = store.DeleteOldJobLogs(ctx, eventRetentionInDays)
			if err != nil {
				return err
			}
			return nil
		})
	return goroutine.NewPeriodicGoroutine(ctx, 60*time.Minute, deleteLogs)
}

func newActionRunner(ctx context.Context, s *cm.Store, metrics codeMonitorsMetrics) *workerutil.Worker {
	options := workerutil.WorkerOptions{
		Name:        "code_monitors_action_jobs_worker",
		NumHandlers: 1,
		Interval:    5 * time.Second,
		Metrics:     metrics.workerMetrics,
	}
	worker := dbworker.NewWorker(ctx, createDBWorkerStoreForActionJobs(s), &actionRunner{s}, options)
	return worker
}

func newActionJobResetter(ctx context.Context, s *cm.Store, metrics codeMonitorsMetrics) *dbworker.Resetter {
	workerStore := createDBWorkerStoreForActionJobs(s)

	options := dbworker.ResetterOptions{
		Name:     "code_monitors_action_jobs_worker_resetter",
		Interval: 1 * time.Minute,
		Metrics: dbworker.ResetterMetrics{
			Errors:              metrics.errors,
			RecordResetFailures: metrics.resetFailures,
			RecordResets:        metrics.resets,
		},
	}
	return dbworker.NewResetter(workerStore, options)
}

func createDBWorkerStoreForTriggerJobs(s *cm.Store) dbworkerstore.Store {
	return dbworkerstore.New(s.Handle(), dbworkerstore.Options{
		Name:              "code_monitors_trigger_jobs_worker_store",
		TableName:         "cm_trigger_jobs",
		ColumnExpressions: cm.TriggerJobsColumns,
		Scan:              cm.ScanTriggerJobs,
		HeartbeatInterval: 15 * time.Second,
		StalledMaxAge:     60 * time.Second,
		RetryAfter:        10 * time.Second,
		MaxNumRetries:     3,
		OrderByExpression: sqlf.Sprintf("id"),
	})
}

func createDBWorkerStoreForActionJobs(s *cm.Store) dbworkerstore.Store {
	return dbworkerstore.New(s.Handle(), dbworkerstore.Options{
		Name:              "code_monitors_action_jobs_worker_store",
		TableName:         "cm_action_jobs",
		ColumnExpressions: cm.ActionJobsColumns,
		Scan:              cm.ScanActionJobs,
		HeartbeatInterval: 15 * time.Second,
		StalledMaxAge:     60 * time.Second,
		RetryAfter:        10 * time.Second,
		MaxNumRetries:     3,
		OrderByExpression: sqlf.Sprintf("id"),
	})
}

type queryRunner struct {
	*cm.Store
}

func (r *queryRunner) Handle(ctx context.Context, record workerutil.Record) (err error) {
	defer func() {
		if err != nil {
			log15.Error("queryRunner.Handle", "error", err)
		}
	}()

	s, err := r.Store.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() { err = s.Done(err) }()

	var q *cm.MonitorQuery
	q, err = s.GetQueryByRecordID(ctx, record.RecordID())
	if err != nil {
		return err
	}
	newQuery := newQueryWithAfterFilter(q)

	// Search.
	var results *gqlSearchResponse
	results, err = search(ctx, newQuery)
	if err != nil {
		return err
	}
	var numResults int
	if results != nil {
		numResults = len(results.Data.Search.Results.Results)
	}
	if numResults > 0 {
		err := s.EnqueueActionEmailsForQueryIDInt64(ctx, q.Id, record.RecordID())
		if err != nil {
			return errors.Errorf("store.EnqueueActionEmailsForQueryIDInt64: %w", err)
		}
	}
	// Log next_run and latest_result to table cm_queries.
	newLatestResult := latestResultTime(q.LatestResult, results, err)
	err = s.SetTriggerQueryNextRun(ctx, q.Id, s.Clock()().Add(5*time.Minute), newLatestResult.UTC())
	if err != nil {
		return err
	}
	// Log the actual query we ran and whether we got any new results.
	err = s.LogSearch(ctx, newQuery, numResults, record.RecordID())
	if err != nil {
		return errors.Errorf("LogSearch: %w", err)
	}
	return nil
}

type actionRunner struct {
	*cm.Store
}

func (r *actionRunner) Handle(ctx context.Context, record workerutil.Record) (err error) {
	log15.Info("actionRunner.Handle starting")
	defer func() {
		if err != nil {
			log15.Error("actionRunner.Handle", "error", err)
		}
	}()

	s, err := r.Store.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() { err = s.Done(err) }()

	var (
		j    *cm.ActionJob
		m    *cm.ActionJobMetadata
		e    *cm.MonitorEmail
		recs []*cm.Recipient
		data *email.TemplateDataNewSearchResults
	)

	var ok bool
	j, ok = record.(*cm.ActionJob)
	if !ok {
		return errors.Errorf("type assertion failed")
	}

	m, err = s.GetActionJobMetadata(ctx, record.RecordID())
	if err != nil {
		return errors.Errorf("store.GetActionJobMetadata: %w", err)
	}

	e, err = s.ActionEmailByIDInt64(ctx, j.Email)
	if err != nil {
		return errors.Errorf("store.ActionEmailByIDInt64: %w", err)
	}

	recs, err = s.AllRecipientsForEmailIDInt64(ctx, j.Email)
	if err != nil {
		return errors.Errorf("store.AllRecipientsForEmailIDInt64: %w", err)
	}

	data, err = email.NewTemplateDataForNewSearchResults(ctx, m.Description, m.Query, e, zeroOrVal(m.NumResults))
	if err != nil {
		return errors.Errorf("email.NewTemplateDataForNewSearchResults: %w", err)
	}
	for _, rec := range recs {
		if rec.NamespaceOrgID != nil {
			// TODO (stefan): Send emails to org members.
			continue
		}
		if rec.NamespaceUserID == nil {
			return errors.Errorf("nil recipient")
		}
		err = email.SendEmailForNewSearchResult(ctx, *rec.NamespaceUserID, data)
		if err != nil {
			return err
		}
	}
	return nil
}

// newQueryWithAfterFilter constructs a new query which finds search results
// introduced after the last time we queried.
func newQueryWithAfterFilter(q *cm.MonitorQuery) string {
	// For q.LatestResult = nil we return a query string without after: filter, which
	// effectively triggers actions immediately provided the query returns any
	// results.
	if q.LatestResult == nil {
		return q.QueryString
	}
	// ATTENTION: This is a stop gap. Add(time.Second) is necessary because currently
	// the after: filter is implemented as "at OR after". If we didn't add a second
	// here, we would send out emails for every run, always showing at least the last
	// result. This means there is non-zero chance that we miss results whenever
	// commits have a timestamp equal to the value of :after but arrive after this
	// job has run.
	afterTime := q.LatestResult.UTC().Add(time.Second).Format(time.RFC3339)
	return strings.Join([]string{q.QueryString, fmt.Sprintf(`after:"%s"`, afterTime)}, " ")
}

func latestResultTime(previousLastResult *time.Time, v *gqlSearchResponse, searchErr error) time.Time {
	if searchErr != nil || len(v.Data.Search.Results.Results) == 0 {
		// Error performing the search, or there were no results. Assume the
		// previous info's result time.
		if previousLastResult != nil {
			return *previousLastResult
		}
		return time.Now()
	}

	// Results are ordered chronologically, so first result is the latest.
	t, err := extractTime(v.Data.Search.Results.Results[0])
	if err != nil {
		// Error already logged by extractTime.
		return time.Now()
	}
	return *t
}

func zeroOrVal(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
