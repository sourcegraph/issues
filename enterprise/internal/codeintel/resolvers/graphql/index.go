package graphql

import (
	"context"
	"strings"

	"github.com/graph-gophers/graphql-go"
	gql "github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	"github.com/sourcegraph/sourcegraph/internal/api"
)

type IndexResolver struct {
	index            store.Index
	locationResolver *CachedLocationResolver
}

func NewIndexResolver(index store.Index, locationResolver *CachedLocationResolver) gql.LSIFIndexResolver {
	return &IndexResolver{
		index:            index,
		locationResolver: locationResolver,
	}
}

func (r *IndexResolver) ID() graphql.ID            { return marshalLSIFIndexGQLID(int64(r.index.ID)) }
func (r *IndexResolver) InputCommit() string       { return r.index.Commit }
func (r *IndexResolver) QueuedAt() gql.DateTime    { return gql.DateTime{Time: r.index.QueuedAt} }
func (r *IndexResolver) State() string             { return strings.ToUpper(r.index.State) }
func (r *IndexResolver) Failure() *string          { return r.index.FailureMessage }
func (r *IndexResolver) StartedAt() *gql.DateTime  { return gql.DateTimeOrNil(r.index.StartedAt) }
func (r *IndexResolver) FinishedAt() *gql.DateTime { return gql.DateTimeOrNil(r.index.FinishedAt) }
func (r *IndexResolver) PlaceInQueue() *int32      { return toInt32(r.index.Rank) }

func (r *IndexResolver) ProjectRoot(ctx context.Context) (*gql.GitTreeEntryResolver, error) {
	return r.locationResolver.Path(ctx, api.RepoID(r.index.RepositoryID), r.index.Commit, "")
}
