package db

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
)

type MockSavedSearches struct {
	ListAll                   func(ctx context.Context) ([]api.SavedQuerySpecAndConfig, error)
	ListSavedSearchesByUserID func(ctx context.Context, userID int32) ([]*types.SavedSearch, error)
	Create                    func(ctx context.Context, newSavedSearch *types.SavedSearch) (*types.SavedSearch, error)
	Update                    func(ctx context.Context, savedSearch *types.SavedSearch) (*types.SavedSearch, error)
	Delete                    func(ctx context.Context, id int32) error
	GetByID                   func(ctx context.Context, id int32) (*api.SavedQuerySpecAndConfig, error)
}
