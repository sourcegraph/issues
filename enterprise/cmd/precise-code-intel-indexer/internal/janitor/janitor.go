package janitor

import (
	"context"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
)

type Janitor struct {
	store   store.Store
	metrics JanitorMetrics
}

var _ goroutine.Handler = &Janitor{}

func New(
	store store.Store,
	janitorInterval time.Duration,
	metrics JanitorMetrics,
) goroutine.BackgroundRoutine {
	return goroutine.NewPeriodicGoroutine(context.Background(), janitorInterval, &Janitor{
		store:   store,
		metrics: metrics,
	})
}

// Handle performs a best-effort cleanup process.
func (j *Janitor) Handle(ctx context.Context) error {
	if err := j.removeRecordsForDeletedRepositories(ctx); err != nil {
		return errors.Wrap(err, "janitor.removeRecordsForDeletedRepositories")
	}

	return nil
}

func (j *Janitor) HandleError(err error) {
	j.metrics.Errors.Inc()
	log15.Error("Failed to run janitor process", "err", err)
}
