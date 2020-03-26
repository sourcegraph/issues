package db

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/authz"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

type MockPerms struct {
	Transact                     func(ctx context.Context) (*PermsStore, error)
	LoadRepoPermissions          func(ctx context.Context, p *authz.RepoPermissions) error
	LoadUserPermissions          func(ctx context.Context, p *authz.UserPermissions) error
	LoadUserPendingPermissions   func(ctx context.Context, p *authz.UserPendingPermissions) error
	SetUserPermissions           func(ctx context.Context, p *authz.UserPermissions) error
	SetRepoPermissions           func(ctx context.Context, p *authz.RepoPermissions) error
	SetRepoPendingPermissions    func(ctx context.Context, accounts *extsvc.ExternalAccounts, p *authz.RepoPermissions) error
	ListPendingUsers             func(ctx context.Context) ([]string, error)
	ListExternalAccounts         func(ctx context.Context, userID int32) ([]*extsvc.ExternalAccount, error)
	GetUserIDsByExternalAccounts func(ctx context.Context, accounts *extsvc.ExternalAccounts) (map[string]int32, error)
}
