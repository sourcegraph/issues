package worker

import (
	"context"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type storeShim struct {
	queueName  string
	queueStore QueueStore
}

type QueueStore interface {
	Dequeue(ctx context.Context, queueName string, payload *executor.Job) (bool, error)
	AddExecutionLogEntry(ctx context.Context, queueName string, jobID int, entry workerutil.ExecutionLogEntry) error
	MarkComplete(ctx context.Context, queueName string, jobID int) error
	MarkErrored(ctx context.Context, queueName string, jobID int, errorMessage string) error
	MarkFailed(ctx context.Context, queueName string, jobID int, errorMessage string) error
}

var _ workerutil.Store = &storeShim{}

func (s *storeShim) QueuedCount(ctx context.Context, extraArguments interface{}) (int, error) {
	return 0, errors.New("unimplemented")
}

func (s *storeShim) Dequeue(ctx context.Context, workerHostname string, extraArguments interface{}) (workerutil.Record, context.CancelFunc, bool, error) {
	var job executor.Job
	dequeued, err := s.queueStore.Dequeue(ctx, s.queueName, &job)
	if err != nil {
		return nil, nil, false, err
	}

	return job, func() {}, dequeued, nil
}

func (s *storeShim) AddExecutionLogEntry(ctx context.Context, id int, entry workerutil.ExecutionLogEntry) error {
	return s.queueStore.AddExecutionLogEntry(ctx, s.queueName, id, entry)
}

func (s *storeShim) MarkComplete(ctx context.Context, id int) (bool, error) {
	return true, s.queueStore.MarkComplete(ctx, s.queueName, id)
}

func (s *storeShim) MarkErrored(ctx context.Context, id int, errorMessage string) (bool, error) {
	return true, s.queueStore.MarkErrored(ctx, s.queueName, id, errorMessage)
}

func (s *storeShim) MarkFailed(ctx context.Context, id int, errorMessage string) (bool, error) {
	return true, s.queueStore.MarkFailed(ctx, s.queueName, id, errorMessage)
}
