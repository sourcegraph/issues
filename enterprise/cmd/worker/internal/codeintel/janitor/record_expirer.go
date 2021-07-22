package janitor

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/goroutine"
)

type recordExpirer struct {
	dbStore DBStore
	ttl     time.Duration
	metrics *metrics
}

var _ goroutine.Handler = &recordExpirer{}

// NewRecordExpirer returns a background routine that periodically removes upload
// and index records that are older than the given TTL. Upload records which have
// valid LSIF data (not just a historic upload failure record) will only be deleted
// if it is not visible at the tip of its repository's default branch.
func NewRecordExpirer(dbStore DBStore, ttl, interval time.Duration, metrics *metrics) goroutine.BackgroundRoutine {
	return goroutine.NewPeriodicGoroutine(context.Background(), interval, &recordExpirer{
		dbStore: dbStore,
		ttl:     ttl,
		metrics: metrics,
	})
}

func (e *recordExpirer) Handle(ctx context.Context) error {
	err1 := e.expireUploads(ctx)
	err2 := e.expireIndexes(ctx)
	if err1 != nil && err2 != nil {
		err1 = multierror.Append(err1, err2)
	}
	if err1 != nil {
		return err1
	}
	return err2
}

func (e *recordExpirer) HandleError(err error) {
	e.metrics.numErrors.Inc()
	log15.Error("Failed to delete old codeintel records", "error", err)
}

func (e *recordExpirer) expireUploads(ctx context.Context) error {
	tx, err := e.dbStore.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() { err = tx.Done(err) }()

	count, err := tx.SoftDeleteOldUploads(ctx, e.ttl, time.Now())
	if err != nil {
		return errors.Wrap(err, "SoftDeleteOldUploads")
	}
	if count > 0 {
		log15.Debug("Deleted old upload records", "count", count)
		e.metrics.numUploadRecordsRemoved.Add(float64(count))
	}

	return nil
}

func (e *recordExpirer) expireIndexes(ctx context.Context) error {
	tx, err := e.dbStore.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() { err = tx.Done(err) }()

	count, err := tx.DeleteOldIndexes(ctx, e.ttl, time.Now())
	if err != nil {
		return errors.Wrap(err, "DeleteOldIndexes")
	}
	if count > 0 {
		log15.Debug("Deleted old index records", "count", count)
		e.metrics.numIndexRecordsRemoved.Add(float64(count))
	}

	return nil
}
