package background

import (
	"time"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
)

// newBatchSpecExecutionResetter creates a dbworker.Resetter that re-enqueues
// lost batch_spec_execution jobs for processing.
func newBatchSpecExecutionResetter(s *store.Store, observationContext *observation.Context, metrics batchChangesMetrics) *dbworker.Resetter {
	// workerStore := NewExecutorStore(s, observationContext)

	options := dbworker.ResetterOptions{
		Name:     "batch_spec_executor_resetter",
		Interval: 1 * time.Minute,
		Metrics:  metrics.executionResetterMetrics,
	}

	resetter := dbworker.NewResetter(nil, options)
	return resetter
}
