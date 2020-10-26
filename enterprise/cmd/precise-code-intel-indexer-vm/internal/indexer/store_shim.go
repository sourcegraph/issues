package indexer

import (
	"context"

	"github.com/pkg/errors"
	queue "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/queue/client"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

// storeShim converts a queue client into a workerutil.Store.
type storeShim struct {
	queueClient queue.Client
}

var _ workerutil.Store = &storeShim{}

// QueuedCount calls into the inner client.
func (s *storeShim) QueuedCount(ctx context.Context, extraArguments interface{}) (int, error) {
	return 0, errors.New("unimplemented")
}

// Dequeue calls into the inner client.
func (s *storeShim) Dequeue(ctx context.Context, extraArguments interface{}) (workerutil.Record, workerutil.Store, bool, error) {
	index, dequeued, err := s.queueClient.Dequeue(ctx)
	return index, s, dequeued, err
}

// Dequeue MarkComplete into the inner client.
func (s *storeShim) MarkComplete(ctx context.Context, id int) (bool, error) {
	return true, s.queueClient.Complete(ctx, id, nil)
}

// MarkErrored calls into the inner client.
func (s *storeShim) MarkErrored(ctx context.Context, id int, failureMessage string) (bool, error) {
	return true, s.queueClient.Complete(ctx, id, errors.New(failureMessage))
}

// Done is a no-op.
func (s *storeShim) Done(err error) error {
	return err
}

// SetLogContents calls into the inner client.
func (s *storeShim) SetLogContents(ctx context.Context, id int, payload string) error {
	return s.queueClient.SetLogContents(ctx, id, payload)
}
