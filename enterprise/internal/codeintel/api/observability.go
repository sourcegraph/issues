package api

import (
	"context"

	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

// An ObservedCodeIntelAPI wraps another CodeIntelAPI with error logging, Prometheus metrics, and tracing.
type ObservedCodeIntelAPI struct {
	codeIntelAPI              CodeIntelAPI
	findClosestDumpsOperation *observation.Operation
	rangesOperation           *observation.Operation
	definitionsOperation      *observation.Operation
	referencesOperation       *observation.Operation
	hoverOperation            *observation.Operation
	diagnosticsOperation      *observation.Operation
}

var _ CodeIntelAPI = &ObservedCodeIntelAPI{}

// NewObservedCodeIntelAPI wraps the given CodeIntelAPI with error logging, Prometheus metrics, and tracing.
func NewObserved(codeIntelAPI CodeIntelAPI, observationContext *observation.Context) CodeIntelAPI {
	metrics := metrics.NewOperationMetrics(
		observationContext.Registerer,
		"code_intel_api",
		metrics.WithLabels("op"),
		metrics.WithCountHelp("Total number of results returned"),
	)

	return &ObservedCodeIntelAPI{
		codeIntelAPI: codeIntelAPI,
		findClosestDumpsOperation: observationContext.Operation(observation.Op{
			Name:         "CodeIntelAPI.FindClosestDumps",
			MetricLabels: []string{"find_closest_dumps"},
			Metrics:      metrics,
		}),
		rangesOperation: observationContext.Operation(observation.Op{
			Name:         "CodeIntelAPI.Ranges",
			MetricLabels: []string{"ranges"},
			Metrics:      metrics,
		}),
		definitionsOperation: observationContext.Operation(observation.Op{
			Name:         "CodeIntelAPI.Definitions",
			MetricLabels: []string{"definitions"},
			Metrics:      metrics,
		}),
		referencesOperation: observationContext.Operation(observation.Op{
			Name:         "CodeIntelAPI.References",
			MetricLabels: []string{"references"},
			Metrics:      metrics,
		}),
		hoverOperation: observationContext.Operation(observation.Op{
			Name:         "CodeIntelAPI.Hover",
			MetricLabels: []string{"hover"},
			Metrics:      metrics,
		}),
		diagnosticsOperation: observationContext.Operation(observation.Op{
			Name:         "CodeIntelAPI.Diagnostics",
			MetricLabels: []string{"diagnostics"},
			Metrics:      metrics,
		}),
	}
}

// FindClosestDumps calls into the inner CodeIntelAPI and registers the observed results.
func (api *ObservedCodeIntelAPI) FindClosestDumps(ctx context.Context, repositoryID int, commit, path string, exactPath bool, indexer string) (dumps []store.Dump, err error) {
	ctx, endObservation := api.findClosestDumpsOperation.With(ctx, &err, observation.Args{})
	defer func() { endObservation(float64(len(dumps)), observation.Args{}) }()
	return api.codeIntelAPI.FindClosestDumps(ctx, repositoryID, commit, path, exactPath, indexer)
}

// Ranges calls into the inner CodeIntelAPI and registers the observed results.
func (api *ObservedCodeIntelAPI) Ranges(ctx context.Context, file string, startLine, endLine, uploadID int) (ranges []ResolvedCodeIntelligenceRange, err error) {
	ctx, endObservation := api.rangesOperation.With(ctx, &err, observation.Args{})
	defer func() { endObservation(float64(len(ranges)), observation.Args{}) }()
	return api.codeIntelAPI.Ranges(ctx, file, startLine, endLine, uploadID)
}

// Definitions calls into the inner CodeIntelAPI and registers the observed results.
func (api *ObservedCodeIntelAPI) Definitions(ctx context.Context, file string, line, character, uploadID int) (definitions []ResolvedLocation, err error) {
	ctx, endObservation := api.definitionsOperation.With(ctx, &err, observation.Args{})
	defer func() { endObservation(float64(len(definitions)), observation.Args{}) }()
	return api.codeIntelAPI.Definitions(ctx, file, line, character, uploadID)
}

// References calls into the inner CodeIntelAPI and registers the observed results.
func (api *ObservedCodeIntelAPI) References(ctx context.Context, repositoryID int, commit string, limit int, cursor Cursor) (references []ResolvedLocation, _ Cursor, _ bool, err error) {
	ctx, endObservation := api.referencesOperation.With(ctx, &err, observation.Args{})
	defer func() { endObservation(float64(len(references)), observation.Args{}) }()
	return api.codeIntelAPI.References(ctx, repositoryID, commit, limit, cursor)
}

// Hover calls into the inner CodeIntelAPI and registers the observed results.
func (api *ObservedCodeIntelAPI) Hover(ctx context.Context, file string, line, character, uploadID int) (_ string, _ bundles.Range, _ bool, err error) {
	ctx, endObservation := api.hoverOperation.With(ctx, &err, observation.Args{})
	defer endObservation(1, observation.Args{})
	return api.codeIntelAPI.Hover(ctx, file, line, character, uploadID)
}

// Diagnostics calls into the inner CodeIntelAPI and registers the observed results.
func (api *ObservedCodeIntelAPI) Diagnostics(ctx context.Context, prefix string, uploadID, limit, offset int) (diagnostics []ResolvedDiagnostic, _ int, err error) {
	ctx, endObservation := api.diagnosticsOperation.With(ctx, &err, observation.Args{})
	defer func() { endObservation(float64(len(diagnostics)), observation.Args{}) }()
	return api.codeIntelAPI.Diagnostics(ctx, prefix, uploadID, limit, offset)
}
