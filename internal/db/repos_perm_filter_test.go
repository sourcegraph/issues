package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/schema"
)

func Test_authzFilter_permissionsUserMapping(t *testing.T) {
	before := globals.PermissionsUserMapping()
	globals.SetPermissionsUserMapping(&schema.PermissionsUserMapping{Enabled: true})
	defer globals.SetPermissionsUserMapping(before)

	tests := []struct {
		name      string
		providers []authz.Provider
		repos     []*types.Repo
		expectErr string
	}{
		// 🚨 SECURITY: We need to make sure the behavior is the same for both "has repos" and "no repos".
		// This is to ensure we always check conflict as the first step.
		{
			name: "site configuration conflict with code host authz providers: has repos",
			providers: []authz.Provider{
				&MockAuthzProvider{
					serviceID:   "https://gitlab.mine/",
					serviceType: extsvc.TypeGitLab,
				},
			},
			repos:     makeRepos("gitlab.mine/u1/r0"),
			expectErr: "The permissions user mapping (site configuration `permissions.userMapping`) cannot be enabled when other authorization providers are in use, please contact site admin to resolve it.",
		},
		{
			name: "site configuration conflict with code host authz providers: no repos",
			providers: []authz.Provider{
				&MockAuthzProvider{
					serviceID:   "https://gitlab.mine/",
					serviceType: extsvc.TypeGitLab,
				},
			},
			repos:     []*types.Repo{},
			expectErr: "The permissions user mapping (site configuration `permissions.userMapping`) cannot be enabled when other authorization providers are in use, please contact site admin to resolve it.",
		},

		{
			name:      "does not allow anonymous access when permissions user mapping is enabled",
			repos:     []*types.Repo{},
			expectErr: "Anonymous access is not allow when permissions user mapping is enabled.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.providers != nil {
				authz.SetProviders(false, test.providers)
				defer authz.SetProviders(true, nil)
			}

			_, err := authzFilter(context.Background(), test.repos, authz.Read)
			if test.expectErr != fmt.Sprintf("%v", err) {
				t.Fatalf("expect error %q but got %q", test.expectErr, err)
			}
		})
	}

	t.Run("called Authz.AuthorizedRepos when permissions user mapping is enabled", func(t *testing.T) {
		user := &types.User{ID: 1}
		Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return user, nil
		}
		ctx := context.Background()
		ctx = actor.WithActor(ctx, &actor.Actor{UID: user.ID})

		calledAuthorizedRepos := false
		Mocks.Authz.AuthorizedRepos = func(_ context.Context, args *AuthorizedReposArgs) ([]*types.Repo, error) {
			calledAuthorizedRepos = true
			if user.ID != args.UserID {
				return nil, fmt.Errorf("args.UserID: want %q but got %q", user.ID, args.UserID)
			}
			return []*types.Repo{}, nil
		}

		_, err := authzFilter(ctx, makeRepos("gitlab.mine/u1/r0"), authz.Read)
		if err != nil {
			t.Fatal(err)
		} else if !calledAuthorizedRepos {
			t.Fatal("!calledAuthorizedRepos")
		}
	})
}

func Test_authzFilter(t *testing.T) {
	publicGitLabRepo := makeRepo("gitlab.mine/user/public", 1, false)
	privateGitLabRepo := makeRepo("gitlab.mine/user/private", 2, true)

	// NOTE: This repository is private but no authz provider is configured for the code host,
	// therefore the access to this repository is not restricted.
	privateGitHubRepo := makeRepo("github.mine/user/private", 3, true)

	t.Run("unauthenticated user should only see public repos", func(t *testing.T) {
		authz.SetProviders(false,
			[]authz.Provider{
				&MockAuthzProvider{
					serviceID:   "https://gitlab.mine/",
					serviceType: extsvc.TypeGitLab,
				},
			},
		)
		defer authz.SetProviders(true, nil)

		repos, err := authzFilter(context.Background(), []*types.Repo{publicGitLabRepo, privateGitLabRepo, privateGitHubRepo}, authz.Read)
		if err != nil {
			t.Fatal(err)
		}

		wantRepos := []*types.Repo{publicGitLabRepo}
		if diff := cmp.Diff(wantRepos, repos); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("authenticated user but no authz providers", func(t *testing.T) {
		user := &types.User{ID: 1}
		Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return user, nil
		}
		defer func() {
			Mocks.Users = MockUsers{}
		}()

		ctx := context.Background()
		ctx = actor.WithActor(ctx, &actor.Actor{UID: user.ID})

		tests := []struct {
			name                string
			authzAllowByDefault bool
			repos               []*types.Repo
			wantRepos           []*types.Repo
		}{
			{
				name:                "authzAllowByDefault=false, only see public repos",
				authzAllowByDefault: false,
				repos:               []*types.Repo{publicGitLabRepo, privateGitLabRepo, privateGitHubRepo},
				wantRepos:           []*types.Repo{publicGitLabRepo},
			},
			{
				name:                "authzAllowByDefault=true, see all repos",
				authzAllowByDefault: true,
				repos:               []*types.Repo{publicGitLabRepo, privateGitLabRepo, privateGitHubRepo},
				wantRepos:           []*types.Repo{publicGitLabRepo, privateGitLabRepo, privateGitHubRepo},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				authz.SetProviders(test.authzAllowByDefault, nil)
				defer authz.SetProviders(true, nil)

				repos, err := authzFilter(ctx, test.repos, authz.Read)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(test.wantRepos, repos); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	t.Run("authenticated user with matching external account", func(t *testing.T) {
		extAccount := extsvc.Account{
			AccountSpec: extsvc.AccountSpec{
				ServiceType: extsvc.TypeGitLab,
				ServiceID:   "https://gitlab.mine/",
				AccountID:   "alice",
			},
		}

		user := &types.User{ID: 1}
		Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return user, nil
		}
		Mocks.ExternalAccounts.List = func(ExternalAccountsListOptions) ([]*extsvc.Account, error) {
			return []*extsvc.Account{&extAccount}, nil
		}
		Mocks.ExternalAccounts.AssociateUserAndSave = func(int32, extsvc.AccountSpec, extsvc.AccountData) error {
			return errors.New("AssociateUserAndSave should not be called")
		}
		Mocks.Authz.GrantPendingPermissions = func(context.Context, *GrantPendingPermissionsArgs) error {
			return errors.New("GrantPendingPermissions should not be called")
		}
		Mocks.Authz.AuthorizedRepos = func(context.Context, *AuthorizedReposArgs) ([]*types.Repo, error) {
			return []*types.Repo{privateGitLabRepo}, nil
		}
		defer func() {
			Mocks.Users = MockUsers{}
			Mocks.ExternalAccounts = MockExternalAccounts{}
			Mocks.Authz = MockAuthz{}
		}()

		ctx := context.Background()
		ctx = actor.WithActor(ctx, &actor.Actor{UID: user.ID})

		tests := []struct {
			name                string
			authzAllowByDefault bool
			repos               []*types.Repo
			wantRepos           []*types.Repo
		}{
			{
				name:                "authzAllowByDefault=false, see explicitly authorized repos",
				authzAllowByDefault: false,
				repos:               []*types.Repo{publicGitLabRepo, privateGitLabRepo, privateGitHubRepo},
				wantRepos:           []*types.Repo{publicGitLabRepo, privateGitLabRepo},
			},
			{
				name:                "authzAllowByDefault=true, see all repos",
				authzAllowByDefault: true,
				repos:               []*types.Repo{publicGitLabRepo, privateGitLabRepo, privateGitHubRepo},
				wantRepos:           []*types.Repo{publicGitLabRepo, privateGitHubRepo, privateGitLabRepo},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				authz.SetProviders(test.authzAllowByDefault,
					[]authz.Provider{
						&MockAuthzProvider{
							serviceID:   "https://gitlab.mine/",
							serviceType: "gitlab",
							okServiceIDs: map[string]struct{}{
								"https://gitlab.mine/": {},
							},
							perms: map[extsvc.Account]map[api.RepoName]authz.Perms{
								extAccount: nil,
							},
						},
					},
				)
				defer authz.SetProviders(true, nil)

				repos, err := authzFilter(ctx, test.repos, authz.Read)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(test.wantRepos, repos); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	t.Run("authenticated user without an external account should fetch and grant", func(t *testing.T) {
		user := &types.User{ID: 1}
		Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return user, nil
		}
		Mocks.ExternalAccounts.List = func(ExternalAccountsListOptions) ([]*extsvc.Account, error) {
			return []*extsvc.Account{
				{
					AccountSpec: extsvc.AccountSpec{
						ServiceType: "gitlab",
						ServiceID:   "https://gitlab.mirror/",
						AccountID:   "alice",
					},
				},
			}, nil
		}

		calledAssociateUserAndSave := false
		callGrantPendingPermissions := false
		Mocks.ExternalAccounts.AssociateUserAndSave = func(int32, extsvc.AccountSpec, extsvc.AccountData) error {
			calledAssociateUserAndSave = true
			return nil
		}
		Mocks.Authz.GrantPendingPermissions = func(context.Context, *GrantPendingPermissionsArgs) error {
			callGrantPendingPermissions = true
			return nil
		}
		Mocks.Authz.AuthorizedRepos = func(context.Context, *AuthorizedReposArgs) ([]*types.Repo, error) {
			return []*types.Repo{privateGitLabRepo}, nil
		}
		defer func() {
			Mocks.Users = MockUsers{}
			Mocks.ExternalAccounts = MockExternalAccounts{}
			Mocks.Authz = MockAuthz{}
		}()

		ctx := context.Background()
		ctx = actor.WithActor(ctx, &actor.Actor{UID: user.ID})

		tests := []struct {
			name                string
			authzAllowByDefault bool
			repos               []*types.Repo
			wantRepos           []*types.Repo
		}{
			{
				name:                "authzAllowByDefault=false, see explicitly authorized repos",
				authzAllowByDefault: false,
				repos:               []*types.Repo{publicGitLabRepo, privateGitLabRepo, privateGitHubRepo},
				wantRepos:           []*types.Repo{publicGitLabRepo, privateGitLabRepo},
			},
			{
				name:                "authzAllowByDefault=true, see all repos",
				authzAllowByDefault: true,
				repos:               []*types.Repo{publicGitLabRepo, privateGitLabRepo, privateGitHubRepo},
				wantRepos:           []*types.Repo{publicGitLabRepo, privateGitHubRepo, privateGitLabRepo},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				authz.SetProviders(test.authzAllowByDefault,
					[]authz.Provider{
						&MockAuthzProvider{
							serviceID:   "https://gitlab.mine/",
							serviceType: "gitlab",
							okServiceIDs: map[string]struct{}{
								"https://gitlab.mirror/": {},
							},
							perms: map[extsvc.Account]map[api.RepoName]authz.Perms{
								{
									AccountSpec: extsvc.AccountSpec{
										ServiceType: "gitlab",
										ServiceID:   "https://gitlab.mine/",
										AccountID:   "alice",
									},
								}: nil,
							},
						},
					},
				)
				defer authz.SetProviders(true, nil)

				repos, err := authzFilter(ctx, test.repos, authz.Read)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(test.wantRepos, repos); diff != "" {
					t.Fatal(diff)
				} else if !calledAssociateUserAndSave {
					t.Fatal("!calledAssociateUserAndSave")
				} else if !callGrantPendingPermissions {
					t.Fatal("!callGrantPendingPermissions")
				}
			})
		}
	})
}

type MockAuthzProvider struct {
	serviceID   string
	serviceType string

	// okServiceIDs indicate services whose external accounts will be straightforwardly translated
	// into external accounts belonging to this provider.
	okServiceIDs map[string]struct{}

	// perms is the map from external user account to repository permissions. The key set must
	// include all user external accounts that are available in this mock instance.
	perms map[extsvc.Account]map[api.RepoName]authz.Perms
}

func (m *MockAuthzProvider) FetchAccount(ctx context.Context, user *types.User, current []*extsvc.Account) (mine *extsvc.Account, err error) {
	if user == nil {
		return nil, nil
	}
	for _, acct := range current {
		if (extsvc.Account{}) == *acct {
			continue
		}
		if _, ok := m.okServiceIDs[acct.ServiceID]; ok {
			myAcct := *acct
			myAcct.ServiceType = m.serviceType
			myAcct.ServiceID = m.serviceID
			if _, acctExistsInPerms := m.perms[myAcct]; acctExistsInPerms {
				return &myAcct, nil
			}
		}
	}
	return nil, nil
}

func (m *MockAuthzProvider) ServiceID() string   { return m.serviceID }
func (m *MockAuthzProvider) ServiceType() string { return m.serviceType }
func (m *MockAuthzProvider) URN() string         { return extsvc.URN(m.serviceType, 0) }
func (m *MockAuthzProvider) Validate() []string  { return nil }

func (m *MockAuthzProvider) FetchUserPerms(context.Context, *extsvc.Account) ([]extsvc.RepoID, error) {
	return nil, nil
}

func (m *MockAuthzProvider) FetchRepoPerms(context.Context, *extsvc.Repository) ([]extsvc.AccountID, error) {
	return nil, nil
}

func makeRepo(name api.RepoName, id api.RepoID, private bool) *types.Repo {
	extName := string(name)
	if extName == "" {
		extName = strconv.Itoa(int(id))
	}

	serviceID, err := url.Parse("https://" + string(name))
	if err != nil {
		panic(err)
	}

	serviceID.Path = "/"
	return &types.Repo{
		ID: id,
		ExternalRepo: api.ExternalRepoSpec{
			ID:          extName,
			ServiceType: "gitlab",
			ServiceID:   serviceID.String(),
		},
		Name:    name,
		Private: private,
	}

}

func makeRepos(names ...api.RepoName) []*types.Repo {
	repos := make([]*types.Repo, len(names))
	for i, name := range names {
		repos[i] = makeRepo(name, api.RepoID(i+1), false)
	}
	return repos
}
