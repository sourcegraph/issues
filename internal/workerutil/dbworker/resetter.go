package dbworker

import (
	"context"
	"errors"
	"time"

	"github.com/efritz/glock"
	"github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

// Resetter periodically moves all unlocked records that have been in the processing state
// for a while back to queued.
//
// An unlocked record signifies that it is not actively being processed and records in this
// state for more than a few seconds are very likely to be stuck after the worker processing
// them has crashed.
type Resetter struct {
	store    store.Store
	options  ResetterOptions
	clock    glock.Clock
	ctx      context.Context // root context passed to the database
	cancel   func()          // cancels the root context
	finished chan struct{}   // signals that Start has finished
}

type ResetterOptions struct {
	Name     string
	Interval time.Duration
	Metrics  ResetterMetrics
}

type ResetterMetrics struct {
	RecordResets        prometheus.Counter
	RecordResetFailures prometheus.Counter
	Errors              prometheus.Counter
}

func NewResetter(store store.Store, options ResetterOptions) *Resetter {
	return newResetter(store, options, glock.NewRealClock())
}

func newResetter(store store.Store, options ResetterOptions, clock glock.Clock) *Resetter {
	ctx, cancel := context.WithCancel(context.Background())

	return &Resetter{
		store:    store,
		options:  options,
		clock:    clock,
		ctx:      ctx,
		cancel:   cancel,
		finished: make(chan struct{}),
	}
}

// Start begins periodically calling reset stalled on the underlying store.
func (r *Resetter) Start() {
	defer close(r.finished)

loop:
	for {
		resetIDs, erroredIDs, err := r.store.ResetStalled(r.ctx)
		if err != nil {
			// If the error is due to the loop being shut down, just break
			for ex := err; ex != nil; ex = errors.Unwrap(ex) {
				if err == r.ctx.Err() {
					break loop
				}
			}

			r.options.Metrics.Errors.Inc()
			log15.Error("Failed to reset stalled records", "name", r.options.Name, "error", err)
		}

		for _, id := range resetIDs {
			log15.Debug("Reset stalled record", "name", r.options.Name, "id", id)
		}
		for _, id := range erroredIDs {
			log15.Debug("Reset stalled record", "name", r.options.Name, "id", id)
		}

		r.options.Metrics.RecordResets.Add(float64(len(resetIDs)))
		r.options.Metrics.RecordResetFailures.Add(float64(len(erroredIDs)))

		select {
		case <-r.clock.After(r.options.Interval):
		case <-r.ctx.Done():
			return
		}
	}
}

// Stop will cause the resetter loop to exit after the current iteration.
func (r *Resetter) Stop() {
	r.cancel()
	<-r.finished
}
