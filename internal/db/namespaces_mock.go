package db

import "context"

type MockNamespaces struct {
	GetByID   func(ctx context.Context, orgID, userID int32) (*Namespace, error)
	GetByName func(ctx context.Context, name string) (*Namespace, error)
}
