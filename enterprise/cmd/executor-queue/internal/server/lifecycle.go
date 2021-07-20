package server

import (
	"context"

	"github.com/hashicorp/go-multierror"
)

// heartbeat will release the transaction for any job that is not confirmed to be in-progress
// by the given executor. This method is called when the executor POSTs its in-progress job
// identifiers to the /heartbeat route. This method returns the set of identifiers which the
// executor erroneously claims to hold (and are sent back as a hint to stop processing).
func (h *handler) heartbeat(ctx context.Context, executorName string, jobIDs []int) ([]int, error) {
	unknownIDs := h.unknownJobs(executorName, jobIDs)
	deadJobs := h.pruneJobs(executorName, jobIDs)
	err := h.requeueJobs(ctx, deadJobs)
	return unknownIDs, err
}

// cleanup will release the transactions held by any executor that has not sent a heartbeat
// in a while. This method is called periodically in the background.
func (h *handler) cleanup(ctx context.Context) error {
	return h.requeueJobs(ctx, h.pruneExecutors())
}

// shutdown releases all transactions. This method is called on process shutdown.
func (h *handler) shutdown() {
	h.m.Lock()
	defer h.m.Unlock()

	for _, executor := range h.executors {
		for _, job := range executor.jobs {
			job.cancel()
		}
	}
}

// unknownJobs returns the set of job identifiers reported by the executor which do not
// have an associated transaction held by this instance of the executor queue. This can
// occur when the executor-queue restarts and loses its transaction state. We send these
// identifiers back to the executor as a hint to stop processing.
func (h *handler) unknownJobs(executorName string, ids []int) []int {
	h.m.Lock()
	defer h.m.Unlock()

	executor, ok := h.executors[executorName]
	if !ok {
		// If executor is unknown, all ids are unknown
		return ids
	}

	idMap := map[int]struct{}{}
	for _, job := range executor.jobs {
		idMap[job.record.RecordID()] = struct{}{}
	}

	unknown := make([]int, 0, len(ids))
	for _, id := range ids {
		if _, ok := idMap[id]; !ok {
			unknown = append(unknown, id)
		}
	}

	return unknown
}

// pruneJobs updates the set of job identifiers assigned to the given executor and returns
// any job that was known to us but not reported by the executor.
func (h *handler) pruneJobs(executorName string, ids []int) (dead []jobMeta) {
	now := h.clock.Now()

	idMap := map[int]struct{}{}
	for _, id := range ids {
		idMap[id] = struct{}{}
	}

	h.m.Lock()
	defer h.m.Unlock()

	executor, ok := h.executors[executorName]
	if !ok {
		executor = &executorMeta{}
		h.executors[executorName] = executor
	}

	var live []jobMeta
	for _, job := range executor.jobs {
		if _, ok := idMap[job.record.RecordID()]; ok || now.Sub(job.started) < h.options.UnreportedMaxAge {
			live = append(live, job)
		} else {
			dead = append(dead, job)
		}
	}

	executor.jobs = live
	executor.lastUpdate = now
	return dead
}

// pruneExecutors will release the transactions held by any executor that has not sent a
// heartbeat in a while and return the attached jobs.
func (h *handler) pruneExecutors() (jobs []jobMeta) {
	h.m.Lock()
	defer h.m.Unlock()

	for name, executor := range h.executors {
		if h.clock.Now().Sub(executor.lastUpdate) <= h.options.DeathThreshold {
			continue
		}

		jobs = append(jobs, executor.jobs...)
		delete(h.executors, name)
	}

	return jobs
}

// requeueJobs releases and requeues each of the given jobs.
func (h *handler) requeueJobs(ctx context.Context, jobs []jobMeta) (errs error) {
	for _, job := range jobs {
		if err := h.requeueJob(ctx, job); err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	return errs
}

// requeueJob requeues the given job and releases the associated transaction.
func (h *handler) requeueJob(ctx context.Context, job jobMeta) error {
	queueOptions, ok := h.options.QueueOptions[job.queueName]
	if !ok {
		return ErrUnknownQueue
	}

	defer func() { h.dequeueSemaphore <- struct{}{} }()
	defer job.cancel()
	return queueOptions.Store.Requeue(ctx, job.record.RecordID(), h.clock.Now().Add(h.options.RequeueDelay))
}
