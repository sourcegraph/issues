package db

import (
	"context"

	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

type MockPerms struct {
	Transact                     func(ctx context.Context) (*PermsStore, error)
	LoadRepoPermissions          func(ctx context.Context, p *authz.RepoPermissions) error
	LoadUserPermissions          func(ctx context.Context, p *authz.UserPermissions) error
	LoadUserPendingPermissions   func(ctx context.Context, p *authz.UserPendingPermissions) error
	SetUserPermissions           func(ctx context.Context, p *authz.UserPermissions) error
	SetRepoPermissions           func(ctx context.Context, p *authz.RepoPermissions) error
	SetRepoPendingPermissions    func(ctx context.Context, accounts *extsvc.Accounts, p *authz.RepoPermissions) error
	TouchRepoPermissions         func(ctx context.Context, repoID int32) (err error)
	ListPendingUsers             func(ctx context.Context) ([]string, error)
	ListExternalAccounts         func(ctx context.Context, userID int32) ([]*extsvc.Account, error)
	GetUserIDsByExternalAccounts func(ctx context.Context, accounts *extsvc.Accounts) (map[string]int32, error)
}
