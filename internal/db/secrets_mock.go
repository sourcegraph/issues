package db

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
)

type MockSecrets struct {
	DeleteByID      func(ctx context.Context, id int32) error
	DeleteByKeyName func(ctx context.Context, keyName string) error
	DeleteBySource  func(ctx context.Context, sourceType string, sourceID int32) error
	GetByID         func(ctx context.Context, id int32) (*types.Secret, error)
	GetByKeyName    func(ctx context.Context, keyName string) (*types.Secret, error)
	GetBySource     func(ctx context.Context, sourceType string, sourceID int32) (*types.Secret, error)
	UpdateByID      func(ctx context.Context, id int32) error
	UpdateByKeyname func(ctx context.Context, keyName string) error
	UpdateBySource  func(ctx context.Context, sourceType string, sourceID int32) error
}
