package resolvers

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

// DocumentationDefinitions returns the list of source locations that define the symbol found at
// the given documentation path ID, if any.
func (r *queryResolver) DocumentationDefinitions(ctx context.Context, pathID string) (_ []AdjustedLocation, err error) {
	ctx, traceLog, endObservation := observeResolver(ctx, &err, "DocumentationDefinitions", r.operations.definitions, slowDefinitionsRequestThreshold, observation.Args{
		LogFields: []log.Field{
			log.Int("repositoryID", r.repositoryID),
			log.String("commit", r.commit),
			log.Int("numUploads", len(r.uploads)),
			log.String("uploads", uploadIDsToString(r.uploads)),
			log.String("pathID", pathID),
		},
	})
	defer endObservation()

	// Because a documentation path ID is repo-local, i.e. the associated definition is always
	// going to be found in the "local" bundle, i.e. it's not possible for it to be in another
	// repository.
	for _, upload := range r.uploads {
		traceLog(log.Int("uploadID", upload.ID))
		locations, _, err := r.lsifStore.DocumentationDefinitions(ctx, upload.ID, pathID, DefinitionsLimit, 0)
		if err != nil {
			return nil, errors.Wrap(err, "lsifStore.DocumentationDefinitions")
		}
		if len(locations) > 0 {
			r.path = locations[0].Path
			uploadsByID := map[int]dbstore.Dump{upload.ID: upload}
			return r.adjustLocations(ctx, uploadsByID, locations)
		}
	}
	return nil, nil
}
