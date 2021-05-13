package graphqlbackend

import (
	"context"
	"errors"

	"github.com/graph-gophers/graphql-go"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/database"
)

func (r *schemaResolver) SetTag(ctx context.Context, args *struct {
	Node    graphql.ID
	Tag     string
	Present bool
}) (*EmptyResponse, error) {
	// 🚨 SECURITY: Only site admins may set tags.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	node, err := r.nodeByID(ctx, args.Node)
	if err != nil {
		return nil, err
	}

	switch node := node.(type) {
	case *UserResolver:
		if err := database.Users(r.db).SetTag(ctx, node.user.ID, args.Tag, args.Present); err != nil {
			return nil, err
		}
		return &EmptyResponse{}, nil

	case *RepositoryResolver:
		if args.Present {
			if _, err := database.RepoTags(r.db).Create(ctx, int(node.IDInt32()), args.Tag); err != nil {
				return nil, err
			}
		} else {
			tag, err := database.RepoTags(r.db).GetByRepoAndTag(ctx, int(node.IDInt32()), args.Tag)
			if err != nil {
				return nil, err
			}

			if err := database.RepoTags(r.db).Delete(ctx, tag); err != nil {
				return nil, err
			}
		}
		return &EmptyResponse{}, nil

	default:
		return nil, errors.New("setting tags is only supported for users")
	}
}
