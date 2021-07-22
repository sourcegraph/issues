package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/derision-test/glock"
	"github.com/inconshreveable/log15"

	apiclient "github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

type handler struct {
	options      Options
	queueOptions QueueOptions
	clock        glock.Clock
	m            sync.Mutex // protects executors
	queueMetrics *QueueMetrics
}

type Options struct {
	// Port is the port on which to listen for HTTP connections.
	Port int

	// RequeueDelay controls how far into the future to make a job record visible to the job
	// queue once the currently processing executor has become unresponsive.
	RequeueDelay time.Duration

	// UnreportedMaxAge is the maximum time between a record being dequeued and it appearing
	// in the executor's heartbeat requests before it being considered lost.
	// TODO: Heartbeat should've solved that. When no heartbeat appears in that timeframe, it is considered lost.
	// UnreportedMaxAge time.Duration

	// DeathThreshold is the minimum time since the last heartbeat of an executor before that
	// executor can be considered as unresponsive. This should be configured to be longer than
	// the duration between heartbeat interval.
	// TODO: Not required anymore, the heartbeat does this now.
	// DeathThreshold time.Duration

	// CleanupInterval is the duration between periodic invocations of Cleanup, which will
	// requeue any records that are "lost" according to the thresholds described above.
	// TODO: Not required anymore, the heartbeat should do this for us.
	// CleanupInterval time.Duration
}

type QueueStore interface {
	store.Store

	ExecutorLastUpdate(ctx context.Context, executorName string) (time.Time, error)
	RecordStartedAt(ctx context.Context, executorName string, recordID int) (time.Time, error)
	HeartbeatRecords(ctx context.Context, executorName string, recordIDs []int) ([]int, error)
}

type QueueOptions struct {
	Name string

	// Store is a required dbworker store store for each registered queue.
	Store QueueStore

	// RecordTransformer is a required hook for each registered queue that transforms a generic
	// record from that queue into the job to be given to an executor.
	RecordTransformer func(ctx context.Context, record workerutil.Record) (apiclient.Job, error)
}

func newHandler(options Options, queueOptions QueueOptions, clock glock.Clock) *handler {
	return newHandlerWithMetrics(options, queueOptions, clock, newQueueMetrics(&observation.TestContext))
}

func newHandlerWithMetrics(options Options, queueOptions QueueOptions, clock glock.Clock, queueMetrics *QueueMetrics) *handler {
	return &handler{
		queueOptions: queueOptions,
		options:      options,
		clock:        clock,
		queueMetrics: queueMetrics,
	}
}

var (
	ErrUnknownQueue = errors.New("unknown queue")
	ErrUnknownJob   = errors.New("unknown job")
)

// dequeue selects a job record from the database and stashes metadata including
// the job record and the locking transaction. If no job is available for processing,
// or the server has hit its maximum transactions, a false-valued flag is returned.
func (m *handler) dequeue(ctx context.Context, executorName, executorHostname string) (_ apiclient.Job, dequeued bool, _ error) {
	record, dequeued, err := m.queueOptions.Store.Dequeue(context.Background(), executorName, nil)
	if err != nil {
		return apiclient.Job{}, false, err
	}
	if !dequeued {
		return apiclient.Job{}, false, nil
	}

	job, err := m.queueOptions.RecordTransformer(ctx, record)
	if err != nil {
		if _, err := m.queueOptions.Store.MarkFailed(ctx, record.RecordID(), fmt.Sprintf("failed to transform record: %s", err)); err != nil {
			log15.Error("Failed to mark record as failed", "recordID", record.RecordID(), "error", err)
		}

		return apiclient.Job{}, false, err
	}

	return job, true, nil
}

// addExecutionLogEntry calls AddExecutionLogEntry for the given job. If the job identifier
// is not known, a false-valued flag is returned.
// TODO: Validate is owned by the executor with executorName
func (m *handler) addExecutionLogEntry(ctx context.Context, executorName string, jobID int, entry workerutil.ExecutionLogEntry) error {
	return m.queueOptions.Store.AddExecutionLogEntry(ctx, jobID, entry)
}

// markComplete calls MarkComplete for the given job, then commits the job's transaction.
// The job is removed from the executor's job list on success.
// TODO: Validate is owned by the executor with executorName
func (m *handler) markComplete(ctx context.Context, executorName string, jobID int) error {
	_, err := m.queueOptions.Store.MarkComplete(ctx, jobID)
	return err
}

// markErrored calls MarkErrored for the given job, then commits the job's transaction.
// The job is removed from the executor's job list on success.
// TODO: Validate is owned by the executor with executorName
func (m *handler) markErrored(ctx context.Context, executorName string, jobID int, errorMessage string) error {
	_, err := m.queueOptions.Store.MarkErrored(ctx, jobID, errorMessage)
	return err
}

// markFailed calls MarkFailed for the given job, then commits the job's transaction.
// The job is removed from the executor's job list on success.
// TODO: Validate is owned by the executor with executorName
func (m *handler) markFailed(ctx context.Context, executorName string, jobID int, errorMessage string) error {
	_, err := m.queueOptions.Store.MarkFailed(ctx, int(jobID), errorMessage)
	return err
}

func (m *handler) updateMetrics() {
	type queueStat struct {
		JobIDs        []int
		ExecutorNames map[string]struct{}
	}
	queueStats := map[string]queueStat{}

	// TODO: reintroduce metrics
	// for executorName, meta := range m.executors {
	// 	for job := range meta.jobs {
	// 		stat, ok := queueStats[m.queueOptions.Name]
	// 		if !ok {
	// 			stat = queueStat{
	// 				ExecutorNames: map[string]struct{}{},
	// 			}
	// 		}

	// 		stat.JobIDs = append(stat.JobIDs, int(job))
	// 		stat.ExecutorNames[executorName] = struct{}{}
	// 		queueStats[m.queueOptions.Name] = stat
	// 	}
	// }

	for queueName, temp := range queueStats {
		m.queueMetrics.NumJobs.WithLabelValues(queueName).Set(float64(len(temp.JobIDs)))
		m.queueMetrics.NumExecutors.WithLabelValues(queueName).Set(float64(len(temp.ExecutorNames)))
	}
}
