package resolvers

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/internal/observation"
)

// DocumentationReferences returns the list of source locations that reference the symbol found at
// the given documentation path ID, if any.
func (r *queryResolver) DocumentationReferences(ctx context.Context, pathID string, limit int, rawCursor string) (_ []AdjustedLocation, _ string, err error) {
	ctx, traceLog, endObservation := observeResolver(ctx, &err, "DocumentationReferences", r.operations.documentationReferences, slowReferencesRequestThreshold, observation.Args{
		LogFields: []log.Field{
			log.Int("repositoryID", r.repositoryID),
			log.String("commit", r.commit),
			log.Int("numUploads", len(r.uploads)),
			log.String("uploads", uploadIDsToString(r.uploads)),
			log.String("pathID", pathID),
		},
	})
	defer endObservation()

	// A documentation path ID is repo-local, i.e. the associated definition is always going to be
	// found in the "local" bundles. It's not possible for it to be in another repository. However,
	// references to that definition could be in other repositories and we want to include those.
	// That requires some fairly complex logic (see query_references.go) for moniker searches, etc.
	//
	// What we do here is first resolve the local definitions, then execute a standard references
	// request on the first location we find.
	for _, upload := range r.uploads {
		traceLog(log.Int("uploadID", upload.ID))
		locations, _, err := r.lsifStore.DocumentationDefinitions(ctx, upload.ID, pathID, DefinitionsLimit, 0)
		if err != nil {
			return nil, "", errors.Wrap(err, "lsifStore.DocumentationDefinitions")
		}
		if len(locations) > 0 {
			location := locations[0]
			r.path = location.Path
			return r.References(ctx, location.Range.Start.Line, location.Range.Start.Character, limit, rawCursor)
		}
	}
	return nil, "", nil
}
