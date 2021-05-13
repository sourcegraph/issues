package graphqlbackend

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

type metadataTagsArgs struct {
	First int32
	After *string
	Tag   *string
}

func (r *RepositoryResolver) MetadataTags(ctx context.Context, args *metadataTagsArgs) (*repositoryMetadataTagConnectionResolver, error) {
	lo := database.LimitOffset{Limit: int(args.First)}
	if args.After != nil {
		after, err := strconv.Atoi(*args.After)
		if err != nil {
			return nil, err
		}
		lo.Offset = after
	}

	opts := database.RepoTagsStoreListOpts{
		LimitOffset: &lo,
		RepoIDs:     []int{int(r.IDInt32())},
	}
	if args.Tag != nil {
		term := *args.Tag
		not := false
		if term[0] == '-' {
			term = term[1:]
			not = true
		}

		opts.Tags = []database.RepoTagsStoreListSearchTerm{
			{Term: term, Not: not},
		}
	}

	return &repositoryMetadataTagConnectionResolver{
		db:   r.db,
		opts: opts,
	}, nil
}

type repositoryMetadataTagConnectionResolver struct {
	db   dbutil.DB
	opts database.RepoTagsStoreListOpts

	once sync.Once
	tags []*types.RepoTag
	next int
	err  error
}

func (r *repositoryMetadataTagConnectionResolver) Nodes(ctx context.Context) ([]*repositoryMetadataTagResolver, error) {
	tags, _, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}

	resolvers := make([]*repositoryMetadataTagResolver, 0, len(tags))
	for _, tag := range tags {
		resolvers = append(resolvers, &repositoryMetadataTagResolver{
			db:  r.db,
			tag: tag,
		})
	}

	return resolvers, nil
}

func (r *repositoryMetadataTagConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	count, err := database.RepoTags(r.db).Count(ctx, r.opts)
	return int32(count), err
}

func (r *repositoryMetadataTagConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	_, next, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}
	if next != 0 {
		return graphqlutil.NextPageCursor(strconv.Itoa(next)), nil
	}
	return graphqlutil.HasNextPage(false), nil
}

func (r *repositoryMetadataTagConnectionResolver) compute(ctx context.Context) ([]*types.RepoTag, int, error) {
	r.once.Do(func() {
		r.tags, r.next, r.err = database.RepoTags(r.db).List(ctx, r.opts)
	})
	return r.tags, r.next, r.err
}

type repositoryMetadataTagResolver struct {
	db  dbutil.DB
	tag *types.RepoTag
}

const repositoryMetadataTagIDKind = "RepositoryMetadataTag"

func (r *repositoryMetadataTagResolver) ID() graphql.ID {
	return relay.MarshalID(repositoryMetadataTagIDKind, r.tag.ID)
}

func (r *repositoryMetadataTagResolver) Repository(ctx context.Context) (*RepositoryResolver, error) {
	return RepositoryByIDInt32(ctx, r.db, api.RepoID(r.tag.RepoID))
}

func (r *repositoryMetadataTagResolver) Tag() string {
	return r.tag.Tag
}

func (r *repositoryMetadataTagResolver) CreatedAt() DateTime {
	return DateTime{Time: r.tag.CreatedAt}
}

func (r *repositoryMetadataTagResolver) UpdatedAt() DateTime {
	return DateTime{Time: r.tag.UpdatedAt}
}

func unmarshalRepositoryMetadataTagID(id graphql.ID) (tagID int64, err error) {
	if kind := relay.UnmarshalKind(id); kind != repositoryMetadataTagIDKind {
		return 0, fmt.Errorf("expected graphql ID to have kind %q; got %q", repositoryMetadataTagIDKind, kind)
	}
	err = relay.UnmarshalSpec(id, &tagID)
	return
}

func (r *schemaResolver) RepositoryMetadataTagByID(ctx context.Context, id graphql.ID) (*repositoryMetadataTagResolver, error) {
	tagID, err := unmarshalRepositoryMetadataTagID(id)
	if err != nil {
		return nil, err
	}

	tag, err := database.RepoTags(r.db).GetByID(ctx, tagID)
	if err != nil {
		return nil, err
	}

	return &repositoryMetadataTagResolver{
		db:  r.db,
		tag: tag,
	}, nil
}
