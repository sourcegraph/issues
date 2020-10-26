package db

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestAuthzStore_GrantPendingPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	// Create user with initially verified email
	user, err := db.Users.Create(ctx, db.NewUser{
		Email:           "alice@example.com",
		Username:        "alice",
		EmailIsVerified: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	code := "verify-code"

	// Add and verify the second email
	err = db.UserEmails.Add(ctx, user.ID, "alice2@example.com", &code)
	if err != nil {
		t.Fatal(err)
	}
	err = db.UserEmails.SetVerified(ctx, user.ID, "alice2@example.com", true)
	if err != nil {
		t.Fatal(err)
	}

	// Add third email and leave as unverified
	err = db.UserEmails.Add(ctx, user.ID, "alice3@example.com", &code)
	if err != nil {
		t.Fatal(err)
	}

	// Add two external accounts
	err = db.ExternalAccounts.AssociateUserAndSave(ctx, user.ID,
		extsvc.AccountSpec{
			ServiceType: "gitlab",
			ServiceID:   "https://gitlab.com/",
			AccountID:   "alice_gitlab",
		},
		extsvc.AccountData{},
	)
	if err != nil {
		t.Fatal(err)
	}
	err = db.ExternalAccounts.AssociateUserAndSave(ctx, user.ID,
		extsvc.AccountSpec{
			ServiceType: "github",
			ServiceID:   "https://github.com/",
			AccountID:   "alice_github",
		},
		extsvc.AccountData{},
	)
	if err != nil {
		t.Fatal(err)
	}

	s := NewAuthzStore(dbconn.Global, clock).(*authzStore)

	// Each update corresponds to a SetRepoPendingPermssions call
	type update struct {
		accounts *extsvc.Accounts
		repoID   int32
	}
	tests := []struct {
		name          string
		config        *schema.PermissionsUserMapping
		args          *db.GrantPendingPermissionsArgs
		updates       []update
		expectRepoIDs []int
	}{
		{
			name: "grant by emails",
			config: &schema.PermissionsUserMapping{
				BindID: "email",
			},
			args: &db.GrantPendingPermissionsArgs{
				UserID: user.ID,
				Perm:   authz.Read,
				Type:   authz.PermRepos,
			},
			updates: []update{
				{
					accounts: &extsvc.Accounts{
						ServiceType: authz.SourcegraphServiceType,
						ServiceID:   authz.SourcegraphServiceID,
						AccountIDs:  []string{"alice@example.com"},
					},
					repoID: 1,
				}, {
					accounts: &extsvc.Accounts{
						ServiceType: authz.SourcegraphServiceType,
						ServiceID:   authz.SourcegraphServiceID,
						AccountIDs:  []string{"alice2@example.com"},
					},
					repoID: 2,
				}, {
					accounts: &extsvc.Accounts{
						ServiceType: authz.SourcegraphServiceType,
						ServiceID:   authz.SourcegraphServiceID,
						AccountIDs:  []string{"alice3@example.com"},
					},
					repoID: 3,
				},
			},
			expectRepoIDs: []int{1, 2},
		},
		{
			name: "grant by username",
			config: &schema.PermissionsUserMapping{
				BindID: "username",
			},
			args: &db.GrantPendingPermissionsArgs{
				UserID: user.ID,
				Perm:   authz.Read,
				Type:   authz.PermRepos,
			},
			updates: []update{
				{
					accounts: &extsvc.Accounts{
						ServiceType: authz.SourcegraphServiceType,
						ServiceID:   authz.SourcegraphServiceID,
						AccountIDs:  []string{"alice"},
					},
					repoID: 1,
				}, {
					accounts: &extsvc.Accounts{
						ServiceType: authz.SourcegraphServiceType,
						ServiceID:   authz.SourcegraphServiceID,
						AccountIDs:  []string{"bob"},
					},
					repoID: 2,
				},
			},
			expectRepoIDs: []int{1},
		},
		{
			name: "grant by external accounts",
			config: &schema.PermissionsUserMapping{
				BindID: "username",
			},
			args: &db.GrantPendingPermissionsArgs{
				UserID: user.ID,
				Perm:   authz.Read,
				Type:   authz.PermRepos,
			},
			updates: []update{
				{
					accounts: &extsvc.Accounts{
						ServiceType: "github",
						ServiceID:   "https://github.com/",
						AccountIDs:  []string{"alice_github"},
					},
					repoID: 1,
				}, {
					accounts: &extsvc.Accounts{
						ServiceType: "gitlab",
						ServiceID:   "https://gitlab.com/",
						AccountIDs:  []string{"alice_gitlab"},
					},
					repoID: 2,
				}, {
					accounts: &extsvc.Accounts{
						ServiceType: "bitbucketServer",
						ServiceID:   "https://bitbucketServer.com/",
						AccountIDs:  []string{"alice_bitbucketServer"},
					},
					repoID: 3,
				},
			},
			expectRepoIDs: []int{1, 2},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer cleanupPermsTables(t, s.store)

			globals.SetPermissionsUserMapping(test.config)

			for _, update := range test.updates {
				err := s.store.SetRepoPendingPermissions(ctx, update.accounts, &authz.RepoPermissions{
					RepoID: update.repoID,
					Perm:   authz.Read,
				})
				if err != nil {
					t.Fatal(err)
				}
			}

			err := s.GrantPendingPermissions(ctx, test.args)
			if err != nil {
				t.Fatal(err)
			}

			p := &authz.UserPermissions{
				UserID: user.ID,
				Perm:   authz.Read,
				Type:   authz.PermRepos,
			}
			err = s.store.LoadUserPermissions(ctx, p)
			if err != nil {
				t.Fatal(err)
			}

			equal(t, "p.IDs", test.expectRepoIDs, bitmapToArray(p.IDs))
		})
	}
}

func TestAuthzStore_AuthorizedRepos(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	s := NewAuthzStore(dbconn.Global, clock).(*authzStore)

	type update struct {
		repoID  int32
		userIDs []uint32
	}
	tests := []struct {
		name        string
		args        *db.AuthorizedReposArgs
		updates     []update
		expectRepos []*types.Repo
	}{
		{
			name: "no repos",
			args: &db.AuthorizedReposArgs{},
		},
		{
			name: "has permissions for user=1",
			args: &db.AuthorizedReposArgs{
				Repos: []*types.Repo{
					{ID: 1},
					{ID: 2},
					{ID: 4},
				},
				UserID: 1,
				Perm:   authz.Read,
				Type:   authz.PermRepos,
			},
			updates: []update{
				{
					repoID:  1,
					userIDs: []uint32{1},
				}, {
					repoID:  2,
					userIDs: []uint32{1},
				}, {
					repoID:  3,
					userIDs: []uint32{1},
				},
			},
			expectRepos: []*types.Repo{
				{ID: 1},
				{ID: 2},
			},
		},
		{
			name: "no permissions for user=2",
			args: &db.AuthorizedReposArgs{
				Repos: []*types.Repo{
					{ID: 1},
					{ID: 2},
				},
				UserID: 2,
				Perm:   authz.Read,
				Type:   authz.PermRepos,
			},
			updates: []update{
				{
					repoID:  1,
					userIDs: []uint32{1},
				},
			},
			expectRepos: []*types.Repo{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer cleanupPermsTables(t, s.store)

			for _, update := range test.updates {
				err := s.store.SetRepoPermissions(ctx, &authz.RepoPermissions{
					RepoID:  update.repoID,
					Perm:    authz.Read,
					UserIDs: toBitmap(update.userIDs...),
				})
				if err != nil {
					t.Fatal(err)
				}
			}

			repos, err := s.AuthorizedRepos(ctx, test.args)
			if err != nil {
				t.Fatal(err)
			}

			equal(t, "repos", test.expectRepos, repos)
		})
	}
}

func TestAuthzStore_RevokeUserPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	s := NewAuthzStore(dbconn.Global, clock).(*authzStore)

	// Set both effective and pending permissions for a user
	if err := s.store.SetRepoPermissions(ctx, &authz.RepoPermissions{
		RepoID:  1,
		Perm:    authz.Read,
		UserIDs: toBitmap(1),
	}); err != nil {
		t.Fatal(err)
	}

	accounts := &extsvc.Accounts{
		ServiceType: authz.SourcegraphServiceType,
		ServiceID:   authz.SourcegraphServiceID,
		AccountIDs:  []string{"alice", "alice@example.com"},
	}
	if err := s.store.SetRepoPendingPermissions(ctx, accounts, &authz.RepoPermissions{
		RepoID: 1,
		Perm:   authz.Read,
	}); err != nil {
		t.Fatal(err)
	}

	// Revoke all of them
	if err := s.RevokeUserPermissions(ctx, &db.RevokeUserPermissionsArgs{
		UserID:   1,
		Accounts: []*extsvc.Accounts{accounts},
	}); err != nil {
		t.Fatal(err)
	}

	// The user should not have any permissions now
	err := s.store.LoadUserPermissions(ctx, &authz.UserPermissions{
		UserID: 1,
		Perm:   authz.Read,
		Type:   authz.PermRepos,
	})
	if err != authz.ErrPermsNotFound {
		t.Fatalf("err: want %q but got %v", authz.ErrPermsNotFound, err)
	}

	for _, bindID := range accounts.AccountIDs {
		err = s.store.LoadUserPendingPermissions(ctx, &authz.UserPendingPermissions{
			ServiceType: accounts.ServiceType,
			ServiceID:   accounts.ServiceID,
			BindID:      bindID,
			Perm:        authz.Read,
			Type:        authz.PermRepos,
		})
		if err != authz.ErrPermsNotFound {
			t.Fatalf("[%s] err: want %q but got %v", bindID, authz.ErrPermsNotFound, err)
		}
	}
}
