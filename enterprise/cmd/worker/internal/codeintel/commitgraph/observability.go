package commitgraph

import (
	"context"

	"github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

type operations struct {
	commitUpdate *observation.Operation
}

func newOperations(dbStore DBStore, observationContext *observation.Context) *operations {
	commitUpdate := observationContext.Operation(observation.Op{
		Name: "codeintel.commitUpdater",
		Metrics: metrics.NewOperationMetrics(
			observationContext.Registerer,
			"codeintel_commit_graph_processor",
			metrics.WithCountHelp("Total number of method invocations."),
		),
	})

	observationContext.Registerer.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "src_codeintel_commit_graph_total",
		Help: "Total number of repositories with stale commit graphs.",
	}, func() float64 {
		dirtyRepositories, err := dbStore.DirtyRepositories(context.Background())
		if err != nil {
			log15.Error("Failed to determine number of dirty repositories", "err", err)
		}

		return float64(len(dirtyRepositories))
	}))

	return &operations{
		commitUpdate: commitUpdate,
	}
}
