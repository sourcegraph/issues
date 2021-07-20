package workerutil

import (
	"context"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/derision-test/glock"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/hostname"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

// Worker is a generic consumer of records from the workerutil store.
type Worker struct {
	store            Store
	handler          Handler
	options          WorkerOptions
	clock            glock.Clock
	handlerSemaphore chan struct{}   // tracks available handler slots
	ctx              context.Context // root context passed to the handler
	cancel           func()          // cancels the root context
	wg               sync.WaitGroup  // tracks active handler routines
	finished         chan struct{}   // signals that Start has finished
}

type WorkerOptions struct {
	// Name denotes the name of the worker used to distinguish log messages and
	// emitted metrics. The worker constructor will fail if this field is not
	// supplied.
	Name string

	// WorkerHostname denotes the hostname of the instance/container the worker
	// is running on. If not supplied, it will be derived from either the `HOSTNAME`
	// env var, or else from os.Hostname()
	WorkerHostname string

	// NumHandlers is the maximum number of handlers that can be invoked
	// concurrently. The underlying store will not be queried while the current
	// number of handlers exceeds this value.
	NumHandlers int

	// Interval is the frequency to poll the underlying store for new work.
	Interval time.Duration

	// Metrics configures logging, tracing, and metrics for the work loop.
	Metrics WorkerMetrics
}

func NewWorker(ctx context.Context, store Store, handler Handler, options WorkerOptions) *Worker {
	return newWorker(ctx, store, handler, options, glock.NewRealClock())
}

func newWorker(ctx context.Context, store Store, handler Handler, options WorkerOptions, clock glock.Clock) *Worker {
	if options.Name == "" {
		panic("no name supplied to github.com/sourcegraph/sourcegraph/internal/workerutil:newWorker")
	}

	if options.WorkerHostname == "" {
		options.WorkerHostname = hostname.Get()
	}

	ctx, cancel := context.WithCancel(ctx)

	handlerSemaphore := make(chan struct{}, options.NumHandlers)
	for i := 0; i < options.NumHandlers; i++ {
		handlerSemaphore <- struct{}{}
	}

	return &Worker{
		store:            store,
		handler:          handler,
		options:          options,
		clock:            clock,
		handlerSemaphore: handlerSemaphore,
		ctx:              ctx,
		cancel:           cancel,
		finished:         make(chan struct{}),
	}
}

// Start begins polling for work from the underlying store and processing records.
func (w *Worker) Start() {
	defer close(w.finished)

loop:
	for {
		ok, err := w.dequeueAndHandle()
		if err != nil {
			if w.ctx.Err() != nil && errors.Is(err, w.ctx.Err()) {
				// If the error is due to the loop being shut down, just break
				break loop
			}

			log15.Error("Failed to dequeue and handle record", "name", w.options.Name, "err", err)
		}

		delay := w.options.Interval
		if ok {
			// If we had a successful dequeue, do not wait the poll interval.
			// Just attempt to get another handler routine and process the next
			// unit of work immediately.
			delay = 0
		}

		select {
		case <-w.clock.After(delay):
		case <-w.ctx.Done():
			break loop
		}
	}

	w.wg.Wait()
}

// Stop will cause the worker loop to exit after the current iteration. This is done by canceling the
// context passed to the database and the handler functions (which may cause the currently processing
// unit of work to fail). This method blocks until all handler goroutines have exited.
func (w *Worker) Stop() {
	w.cancel()
	<-w.finished
}

// dequeueAndHandle selects a queued record to process. This method returns false if no such record
// can be dequeued and returns an error only on failure to dequeue a new record - no handler errors
// will bubble up.
func (w *Worker) dequeueAndHandle() (dequeued bool, err error) {
	select {
	// If we block here we are waiting for a handler to exit so that we do not
	// exceed our configured concurrency limit.
	case <-w.handlerSemaphore:
	case <-w.ctx.Done():
		return false, w.ctx.Err()
	}
	defer func() {
		if !dequeued {
			// Ensure that if we do not dequeue a record successfully we do not
			// leak from the semaphore. This will happen if the pre dequeue hook
			// fails, if the dequeue call fails, or if there are no records to
			// process.
			w.handlerSemaphore <- struct{}{}
		}
	}()

	dequeueable, extraDequeueArguments, err := w.preDequeueHook()
	if err != nil {
		return false, errors.Wrap(err, "Handler.PreDequeueHook")
	}
	if !dequeueable {
		// Hook declined to dequeue a record
		return false, nil
	}

	// Select a queued record to process and the transaction that holds it
	record, cancel, dequeued, err := w.store.Dequeue(w.ctx, w.options.WorkerHostname, extraDequeueArguments)
	if err != nil {
		return false, errors.Wrap(err, "store.Dequeue")
	}
	if !dequeued {
		// Nothing to process
		return false, nil
	}

	w.options.Metrics.numJobs.Inc()
	log15.Debug("Dequeued record for processing", "name", w.options.Name, "id", record.RecordID())

	if hook, ok := w.handler.(WithHooks); ok {
		hook.PreHandle(w.ctx, record)
	}

	w.wg.Add(1)

	go func() {
		defer func() {
			if hook, ok := w.handler.(WithHooks); ok {
				hook.PostHandle(w.ctx, record)
			}

			cancel()
			w.options.Metrics.numJobs.Dec()
			w.handlerSemaphore <- struct{}{}
			w.wg.Done()
		}()

		if err := w.handle(record); err != nil {
			log15.Error("Failed to finalize record", "name", w.options.Name, "err", err)
		}
	}()

	return true, nil
}

// handle processes the given record. This method returns an error only if there is an issue updating
// the record to a terminal state - no handler errors will bubble up.
func (w *Worker) handle(record Record) (err error) {
	ctx, endOperation := w.options.Metrics.operations.handle.With(w.ctx, &err, observation.Args{})
	defer endOperation(1, observation.Args{})

	handleErr := w.handler.Handle(ctx, record)

	if errcode.IsNonRetryable(handleErr) {
		if marked, markErr := w.store.MarkFailed(ctx, record.RecordID(), handleErr.Error()); markErr != nil {
			return errors.Wrap(markErr, "store.MarkFailed")
		} else if marked {
			log15.Warn("Marked record as failed", "name", w.options.Name, "id", record.RecordID(), "err", handleErr)
		}
	} else if handleErr != nil {
		if marked, markErr := w.store.MarkErrored(ctx, record.RecordID(), handleErr.Error()); markErr != nil {
			return errors.Wrap(markErr, "store.MarkErrored")
		} else if marked {
			log15.Warn("Marked record as errored", "name", w.options.Name, "id", record.RecordID(), "err", handleErr)
		}
	} else {
		if marked, markErr := w.store.MarkComplete(ctx, record.RecordID()); markErr != nil {
			return errors.Wrap(markErr, "store.MarkComplete")
		} else if marked {
			log15.Debug("Marked record as complete", "name", w.options.Name, "id", record.RecordID())
		}
	}

	log15.Debug("Handled record", "name", w.options.Name, "id", record.RecordID())
	return nil
}

// preDequeueHook invokes the handler's pre-dequeue hook if it exists.
func (w *Worker) preDequeueHook() (dequeueable bool, extraDequeueArguments interface{}, err error) {
	if o, ok := w.handler.(WithPreDequeue); ok {
		return o.PreDequeue(w.ctx)
	}

	return true, nil, nil
}
