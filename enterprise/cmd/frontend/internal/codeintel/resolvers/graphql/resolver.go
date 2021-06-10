package graphql

import (
	"context"
	"strings"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	gql "github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/codeintel/resolvers"
	store "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

const (
	DefaultUploadPageSize = 50
	DefaultIndexPageSize  = 50
)

var errAutoIndexingNotEnabled = errors.New("precise code intelligence auto indexing is not enabled")

// Resolver is the main interface to code intel-related operations exposted to the GraphQL API. This
// resolver concerns itself with GraphQL/API-specific behaviors (auth, validation, marshaling, etc.).
// All code intel-specific behavior is delegated to the underlying resolver instance, which is defined
// in the parent package.
type Resolver struct {
	db               dbutil.DB
	resolver         resolvers.Resolver
	locationResolver *CachedLocationResolver
}

// NewResolver creates a new Resolver with the given resolver that defines all code intel-specific behavior.
func NewResolver(db dbutil.DB, resolver resolvers.Resolver) gql.CodeIntelResolver {
	return &Resolver{
		db:               db,
		resolver:         resolver,
		locationResolver: NewCachedLocationResolver(db),
	}
}

func (r *Resolver) NodeResolvers() map[string]gql.NodeByIDFunc {
	return map[string]gql.NodeByIDFunc{
		"LSIFUpload": func(ctx context.Context, id graphql.ID) (gql.Node, error) {
			return r.LSIFUploadByID(ctx, id)
		},
		"LSIFIndex": func(ctx context.Context, id graphql.ID) (gql.Node, error) {
			return r.LSIFIndexByID(ctx, id)
		},
	}
}

func (r *Resolver) LSIFUploadByID(ctx context.Context, id graphql.ID) (gql.LSIFUploadResolver, error) {
	uploadID, err := unmarshalLSIFUploadGQLID(id)
	if err != nil {
		return nil, err
	}

	// Create a new prefetcher here as we only want to cache upload and index records in
	// the same graphQL request, not across different request.
	prefetcher := NewPrefetcher(r.resolver)

	upload, exists, err := prefetcher.GetUploadByID(ctx, int(uploadID))
	if err != nil || !exists {
		return nil, err
	}

	return NewUploadResolver(upload, prefetcher, r.locationResolver), nil
}

func (r *Resolver) LSIFUploads(ctx context.Context, args *gql.LSIFUploadsQueryArgs) (gql.LSIFUploadConnectionResolver, error) {
	// Delegate behavior to LSIFUploadsByRepo with no specified repository identifier
	return r.LSIFUploadsByRepo(ctx, &gql.LSIFRepositoryUploadsQueryArgs{LSIFUploadsQueryArgs: args})
}

func (r *Resolver) LSIFUploadsByRepo(ctx context.Context, args *gql.LSIFRepositoryUploadsQueryArgs) (gql.LSIFUploadConnectionResolver, error) {
	opts, err := makeGetUploadsOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	// Create a new prefetcher here as we only want to cache upload and index records in
	// the same graphQL request, not across different request.
	prefetcher := NewPrefetcher(r.resolver)

	return NewUploadConnectionResolver(r.resolver.UploadConnectionResolver(opts), prefetcher, r.locationResolver), nil
}

func (r *Resolver) DeleteLSIFUpload(ctx context.Context, args *struct{ ID graphql.ID }) (*gql.EmptyResponse, error) {
	// 🚨 SECURITY: Only site admins may delete LSIF data for now
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return nil, err
	}

	uploadID, err := unmarshalLSIFUploadGQLID(args.ID)
	if err != nil {
		return nil, err
	}

	if err := r.resolver.DeleteUploadByID(ctx, int(uploadID)); err != nil {
		return nil, err
	}

	return &gql.EmptyResponse{}, nil
}

var autoIndexingEnabled = conf.CodeIntelAutoIndexingEnabled

func (r *Resolver) LSIFIndexByID(ctx context.Context, id graphql.ID) (gql.LSIFIndexResolver, error) {
	if !autoIndexingEnabled() {
		return nil, errAutoIndexingNotEnabled
	}

	indexID, err := unmarshalLSIFIndexGQLID(id)
	if err != nil {
		return nil, err
	}

	// Create a new prefetcher here as we only want to cache upload and index records in
	// the same graphQL request, not across different request.
	prefetcher := NewPrefetcher(r.resolver)

	index, exists, err := prefetcher.GetIndexByID(ctx, int(indexID))
	if err != nil || !exists {
		return nil, err
	}

	return NewIndexResolver(index, prefetcher, r.locationResolver), nil
}

func (r *Resolver) LSIFIndexes(ctx context.Context, args *gql.LSIFIndexesQueryArgs) (gql.LSIFIndexConnectionResolver, error) {
	if !autoIndexingEnabled() {
		return nil, errAutoIndexingNotEnabled
	}

	// Delegate behavior to LSIFIndexesByRepo with no specified repository identifier
	return r.LSIFIndexesByRepo(ctx, &gql.LSIFRepositoryIndexesQueryArgs{LSIFIndexesQueryArgs: args})
}

func (r *Resolver) LSIFIndexesByRepo(ctx context.Context, args *gql.LSIFRepositoryIndexesQueryArgs) (gql.LSIFIndexConnectionResolver, error) {
	if !autoIndexingEnabled() {
		return nil, errAutoIndexingNotEnabled
	}

	opts, err := makeGetIndexesOptions(ctx, args)
	if err != nil {
		return nil, err
	}

	// Create a new prefetcher here as we only want to cache upload and index records in
	// the same graphQL request, not across different request.
	prefetcher := NewPrefetcher(r.resolver)

	return NewIndexConnectionResolver(r.resolver.IndexConnectionResolver(opts), prefetcher, r.locationResolver), nil
}

func (r *Resolver) DeleteLSIFIndex(ctx context.Context, args *struct{ ID graphql.ID }) (*gql.EmptyResponse, error) {
	if !autoIndexingEnabled() {
		return nil, errAutoIndexingNotEnabled
	}

	// 🚨 SECURITY: Only site admins may delete LSIF data for now
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return nil, err
	}

	indexID, err := unmarshalLSIFIndexGQLID(args.ID)
	if err != nil {
		return nil, err
	}

	if err := r.resolver.DeleteIndexByID(ctx, int(indexID)); err != nil {
		return nil, err
	}

	return &gql.EmptyResponse{}, nil
}

func (r *Resolver) IndexConfiguration(ctx context.Context, id graphql.ID) (gql.IndexConfigurationResolver, error) {
	if !autoIndexingEnabled() {
		return nil, errAutoIndexingNotEnabled
	}

	repositoryID, err := gql.UnmarshalRepositoryID(id)
	if err != nil {
		return nil, err
	}

	configuration, err := r.resolver.IndexConfiguration(ctx, int(repositoryID))
	if err != nil {
		return nil, err
	}

	return NewIndexConfigurationResolver(configuration), nil
}

func (r *Resolver) UpdateRepositoryIndexConfiguration(ctx context.Context, args *gql.UpdateRepositoryIndexConfigurationArgs) (*gql.EmptyResponse, error) {
	if !autoIndexingEnabled() {
		return nil, errAutoIndexingNotEnabled
	}

	// 🚨 SECURITY: Only site admins may configure indexing jobs for now
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return nil, err
	}

	repositoryID, err := unmarshalLSIFIndexGQLID(args.Repository)
	if err != nil {
		return nil, err
	}

	if err := r.resolver.UpdateIndexConfigurationByRepositoryID(ctx, int(repositoryID), args.Configuration); err != nil {
		return nil, err
	}

	return &gql.EmptyResponse{}, nil
}

func (r *Resolver) CommitGraph(ctx context.Context, id graphql.ID) (gql.CodeIntelligenceCommitGraphResolver, error) {
	repositoryID, err := gql.UnmarshalRepositoryID(id)
	if err != nil {
		return nil, err
	}

	return r.resolver.CommitGraph(ctx, int(repositoryID))
}

func (r *Resolver) QueueAutoIndexJobForRepo(ctx context.Context, args *struct{ Repository graphql.ID }) (*gql.EmptyResponse, error) {
	if !autoIndexingEnabled() {
		return nil, errAutoIndexingNotEnabled
	}

	repositoryID, err := gql.UnmarshalRepositoryID(args.Repository)
	if err != nil {
		return nil, err
	}

	return &gql.EmptyResponse{}, r.resolver.QueueAutoIndexJobForRepo(ctx, int(repositoryID))
}

func (r *Resolver) GitBlobLSIFData(ctx context.Context, args *gql.GitBlobLSIFDataArgs) (gql.GitBlobLSIFDataResolver, error) {
	resolver, err := r.resolver.QueryResolver(ctx, args)
	if err != nil || resolver == nil {
		return nil, err
	}

	return NewQueryResolver(resolver, r.locationResolver), nil
}

// makeGetUploadsOptions translates the given GraphQL arguments into options defined by the
// store.GetUploads operations.
func makeGetUploadsOptions(ctx context.Context, args *gql.LSIFRepositoryUploadsQueryArgs) (store.GetUploadsOptions, error) {
	repositoryID, err := resolveRepositoryID(ctx, args.RepositoryID)
	if err != nil {
		return store.GetUploadsOptions{}, err
	}

	offset, err := decodeIntCursor(args.After)
	if err != nil {
		return store.GetUploadsOptions{}, err
	}

	return store.GetUploadsOptions{
		RepositoryID: repositoryID,
		State:        strings.ToLower(derefString(args.State, "")),
		Term:         derefString(args.Query, ""),
		VisibleAtTip: derefBool(args.IsLatestForRepo, false),
		Limit:        derefInt32(args.First, DefaultUploadPageSize),
		Offset:       offset,
	}, nil
}

// makeGetIndexesOptions translates the given GraphQL arguments into options defined by the
// store.GetIndexes operations.
func makeGetIndexesOptions(ctx context.Context, args *gql.LSIFRepositoryIndexesQueryArgs) (store.GetIndexesOptions, error) {
	repositoryID, err := resolveRepositoryID(ctx, args.RepositoryID)
	if err != nil {
		return store.GetIndexesOptions{}, err
	}

	offset, err := decodeIntCursor(args.After)
	if err != nil {
		return store.GetIndexesOptions{}, err
	}

	return store.GetIndexesOptions{
		RepositoryID: repositoryID,
		State:        strings.ToLower(derefString(args.State, "")),
		Term:         derefString(args.Query, ""),
		Limit:        derefInt32(args.First, DefaultIndexPageSize),
		Offset:       offset,
	}, nil
}

// resolveRepositoryByID gets a repository's internal identifier from a GraphQL identifier.
func resolveRepositoryID(ctx context.Context, id graphql.ID) (int, error) {
	if id == "" {
		return 0, nil
	}

	repoID, err := gql.UnmarshalRepositoryID(id)
	if err != nil {
		return 0, err
	}

	return int(repoID), nil
}
