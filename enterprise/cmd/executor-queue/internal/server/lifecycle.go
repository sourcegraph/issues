package server

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
)

// heartbeat will release the transaction for any job that is not confirmed to be in-progress
// by the given executor. This method is called when the executor POSTs its in-progress job
// identifiers to the /heartbeat route. This method returns the set of identifiers which the
// executor erroneously claims to hold (and are sent back as a hint to stop processing).
func (h *handler) heartbeat(ctx context.Context, executorName string, jobIDs []int) ([]int, error) {
	fmt.Printf("heartbeat jobIDs=%+v\n", jobIDs)
	if len(jobIDs) == 0 {
		return nil, nil
	}

	unknownIDs, err := h.heartbeatJobs(ctx, executorName, jobIDs)
	// err2 := h.requeueJobs(ctx, unknownIDs)
	// if err != nil && err2 != nil {
	// 	err = multierror.Append(err, err2)
	// }
	// if err2 != nil {
	// 	err = err2
	// }
	fmt.Printf("heartbeat unknownIDs=%+v\n", unknownIDs)
	return unknownIDs, err
}

// heartbeatJobs updates the set of job identifiers assigned to the given executor and returns
// any job that was known to us but not reported by the executor, plus the set of job identifiers
// reported by the executor which do not have an associated record held by this instance of the
// executor queue. This can occur when the executor-queue restarts and loses its in-memory state.
// We send these identifiers back to the executor as a hint to stop processing.
func (h *handler) heartbeatJobs(ctx context.Context, executorName string, ids []int) (unknownIDs []int, errs error) {

	h.m.Lock()
	defer h.m.Unlock()

	knownIDs, err := h.queueOptions.Store.HeartbeatRecords(ctx, executorName, ids)
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	knownIDsMap := map[int]struct{}{}
	for _, id := range knownIDs {
		knownIDsMap[id] = struct{}{}
	}

	for _, id := range ids {
		if _, ok := knownIDsMap[id]; !ok {
			unknownIDs = append(unknownIDs, id)
		}
	}

	return unknownIDs, errs
}

// requeueJobs releases and requeues each of the given jobs.
func (h *handler) requeueJobs(ctx context.Context, jobs []int) (errs error) {
	for _, job := range jobs {
		if err := h.requeueJob(ctx, job); err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	return errs
}

// requeueJob requeues the given job and releases the associated transaction.
func (h *handler) requeueJob(ctx context.Context, job int) error {
	return h.queueOptions.Store.Requeue(ctx, int(job), h.clock.Now().Add(h.options.RequeueDelay))
}
