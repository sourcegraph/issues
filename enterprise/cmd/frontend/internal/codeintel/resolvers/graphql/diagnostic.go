package graphql

import (
	"context"
	"fmt"

	gql "github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/codeintel/resolvers"
)

type DiagnosticResolver struct {
	diagnostic       resolvers.AdjustedDiagnostic
	locationResolver *CachedLocationResolver
}

func NewDiagnosticResolver(diagnostic resolvers.AdjustedDiagnostic, locationResolver *CachedLocationResolver) gql.DiagnosticResolver {
	return &DiagnosticResolver{
		diagnostic:       diagnostic,
		locationResolver: locationResolver,
	}
}

func (r *DiagnosticResolver) Severity() (*string, error) { return toSeverity(r.diagnostic.Severity) }
func (r *DiagnosticResolver) Code() (*string, error)     { return strPtr(r.diagnostic.Code), nil }
func (r *DiagnosticResolver) Source() (*string, error)   { return strPtr(r.diagnostic.Source), nil }
func (r *DiagnosticResolver) Message() (*string, error)  { return strPtr(r.diagnostic.Message), nil }
func (r *DiagnosticResolver) Tags() ([]int32, error) {
	tags := make([]int32, len(r.diagnostic.Tags))
	for index, tag := range r.diagnostic.Tags {
		tags[index] = int32(tag)
	}
	return tags, nil
}

func (r *DiagnosticResolver) Location(ctx context.Context) (gql.LocationResolver, error) {
	return resolveLocation(
		ctx,
		r.locationResolver,
		resolvers.AdjustedLocation{
			Dump:           r.diagnostic.Dump,
			Path:           r.diagnostic.Path,
			AdjustedCommit: r.diagnostic.AdjustedCommit,
			AdjustedRange:  r.diagnostic.AdjustedRange,
		},
	)
}

var severities = map[int]string{
	1: "ERROR",
	2: "WARNING",
	3: "INFORMATION",
	4: "HINT",
}

func toSeverity(val int) (*string, error) {
	severity, ok := severities[val]
	if !ok {
		return nil, fmt.Errorf("unknown diagnostic severity %d", val)
	}

	return &severity, nil
}
