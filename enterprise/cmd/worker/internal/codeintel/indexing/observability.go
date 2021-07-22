package indexing

import (
	"fmt"
	"sync"

	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

type operations struct {
	HandleIndexScheduler *observation.Operation
	QueueRepository      *observation.Operation
}

var (
	singletonOperations *operations
	once                sync.Once
)

func newOperations(observationContext *observation.Context) *operations {
	once.Do(func() {
		metrics := metrics.NewOperationMetrics(
			observationContext.Registerer,
			"codeintel_index_scheduler",
			metrics.WithLabels("op"),
			metrics.WithCountHelp("Total number of method invocations."),
		)

		op := func(name string) *observation.Operation {
			return observationContext.Operation(observation.Op{
				Name:         fmt.Sprintf("codeintel.indexing.%s", name),
				MetricLabels: []string{name},
				Metrics:      metrics,
			})
		}

		singletonOperations = &operations{
			HandleIndexScheduler: op("HandleIndexSchedule"),
			QueueRepository:      op("QueueRepository"),
		}
	})
	return singletonOperations
}
