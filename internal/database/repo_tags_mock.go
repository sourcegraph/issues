package database

import (
	"context"

	"github.com/sourcegraph/sourcegraph/internal/types"
)

type MockRepoTags struct {
	Count           func(context.Context, RepoTagsStoreListOpts) (int, error)
	Create          func(context.Context, int, string) (*types.RepoTag, error)
	Delete          func(context.Context, *types.RepoTag) error
	GetByID         func(context.Context, int64) (*types.RepoTag, error)
	GetByRepoAndTag func(context.Context, int, string) (*types.RepoTag, error)
	List            func(context.Context, RepoTagsStoreListOpts) ([]*types.RepoTag, int, error)
	Update          func(context.Context, *types.RepoTag) (*types.RepoTag, error)
}
