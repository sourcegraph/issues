package repos

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestStatusMessages(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	db := dbtest.NewDB(t, "")
	store := NewStore(db, sql.TxOptions{})

	admin, err := database.Users(db).Create(ctx, database.NewUser{
		Email:                 "a1@example.com",
		Username:              "a1",
		Password:              "p",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}
	nonAdmin, err := database.Users(db).Create(ctx, database.NewUser{
		Email:                 "u1@example.com",
		Username:              "u1",
		Password:              "p",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	siteLevelService := &types.ExternalService{
		ID:          1,
		Config:      `{}`,
		Kind:        extsvc.KindGitHub,
		DisplayName: "github.com - site",
	}
	err = database.ExternalServices(db).Upsert(ctx, siteLevelService)
	if err != nil {
		t.Fatal(err)
	}
	userService := &types.ExternalService{
		ID:              2,
		Config:          `{}`,
		Kind:            extsvc.KindGitHub,
		DisplayName:     "github.com - user",
		NamespaceUserID: nonAdmin.ID,
	}
	err = database.ExternalServices(db).Upsert(ctx, userService)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name             string
		stored           types.Repos
		gitserverCloned  []string
		gitserverFailure map[string]bool
		sourcerErr       error
		listRepoErr      error
		res              []StatusMessage
		user             *types.User
		// maps repoName to external service
		repoOwner map[api.RepoName]*types.ExternalService
		err       string
	}{
		{
			name:            "all cloned",
			gitserverCloned: []string{"foobar"},
			stored:          []*types.Repo{{Name: "foobar"}},
			user:            admin,
			res:             nil,
		},
		{
			name:            "nothing cloned",
			stored:          []*types.Repo{{Name: "foobar"}},
			user:            admin,
			gitserverCloned: []string{},
			repoOwner: map[api.RepoName]*types.ExternalService{
				"foobar": siteLevelService,
			},
			res: []StatusMessage{
				{
					Cloning: &CloningProgress{
						Message: "1 repository cloning...",
					},
				},
			},
		},
		{
			name:            "subset cloned",
			stored:          []*types.Repo{{Name: "foobar"}, {Name: "barfoo"}},
			user:            admin,
			gitserverCloned: []string{"foobar"},
			repoOwner: map[api.RepoName]*types.ExternalService{
				"foobar": siteLevelService,
				"barfoo": siteLevelService,
			},
			res: []StatusMessage{
				{
					Cloning: &CloningProgress{
						Message: "1 repository cloning...",
					},
				},
			},
		},
		{
			name:   "non admin users should only count their own non cloned repos",
			stored: []*types.Repo{{Name: "foobar"}, {Name: "barfoo"}},
			repoOwner: map[api.RepoName]*types.ExternalService{
				"foobar": userService,
			},
			user: nonAdmin,
			res: []StatusMessage{
				{
					Cloning: &CloningProgress{
						Message: "1 repository cloning...",
					},
				},
			},
		},
		{
			name:            "more cloned than stored",
			stored:          []*types.Repo{{Name: "foobar"}},
			user:            admin,
			gitserverCloned: []string{"foobar", "barfoo"},
			res:             nil,
		},
		{
			name:            "cloned different than stored",
			stored:          []*types.Repo{{Name: "foobar"}, {Name: "barfoo"}},
			user:            admin,
			gitserverCloned: []string{"one", "two", "three"},
			repoOwner: map[api.RepoName]*types.ExternalService{
				"foobar": siteLevelService,
				"barfoo": siteLevelService,
			},
			res: []StatusMessage{
				{
					Cloning: &CloningProgress{
						Message: "2 repositories cloning...",
					},
				},
			},
		},
		{
			name:             "one repo failed to sync",
			stored:           []*types.Repo{{Name: "foobar"}, {Name: "barfoo"}},
			user:             admin,
			gitserverCloned:  []string{"foobar", "barfoo"},
			gitserverFailure: map[string]bool{"foobar": true},
			repoOwner: map[api.RepoName]*types.ExternalService{
				"foobar": siteLevelService,
				"barfoo": siteLevelService,
			},
			res: []StatusMessage{
				{
					SyncError: &SyncError{
						Message: "1 repository could not be synced",
					},
				},
			},
		},
		{
			name:             "two repos failed to sync",
			stored:           []*types.Repo{{Name: "foobar"}, {Name: "barfoo"}},
			user:             admin,
			gitserverCloned:  []string{"foobar", "barfoo"},
			gitserverFailure: map[string]bool{"foobar": true, "barfoo": true},
			repoOwner: map[api.RepoName]*types.ExternalService{
				"foobar": siteLevelService,
				"barfoo": siteLevelService,
			},
			res: []StatusMessage{
				{
					SyncError: &SyncError{
						Message: "2 repositories could not be synced",
					},
				},
			},
		},
		{
			name:            "case insensitivity",
			gitserverCloned: []string{"foobar"},
			stored:          []*types.Repo{{Name: "FOOBar"}},
			user:            admin,
			res:             nil,
		},
		{
			name:            "case insensitivity to gitserver names",
			gitserverCloned: []string{"FOOBar"},
			stored:          []*types.Repo{{Name: "FOOBar"}},
			user:            admin,
			res:             nil,
		},
		{
			name:       "one external service syncer err",
			user:       admin,
			sourcerErr: errors.New("github is down"),
			res: []StatusMessage{
				{
					ExternalServiceSyncError: &ExternalServiceSyncError{
						Message:           "fetching from code host github.com - site: 1 error occurred:\n\t* github is down\n\n",
						ExternalServiceId: siteLevelService.ID,
					},
				},
			},
		},
		{
			name:        "one syncer err",
			user:        admin,
			listRepoErr: errors.New("could not connect to database"),
			res: []StatusMessage{
				{
					ExternalServiceSyncError: &ExternalServiceSyncError{
						Message:           "syncer.sync.store.list-repos: could not connect to database",
						ExternalServiceId: siteLevelService.ID,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		ctx := context.Background()

		t.Run(tc.name, func(t *testing.T) {
			stored := tc.stored.Clone()
			for _, r := range stored {
				r.ExternalRepo = api.ExternalRepoSpec{
					ID:          uuid.New().String(),
					ServiceType: extsvc.TypeGitHub,
					ServiceID:   "https://github.com/",
				}
			}

			err := database.Repos(db).Create(ctx, stored...)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				ids := make([]api.RepoID, 0, len(stored))
				for _, r := range stored {
					ids = append(ids, r.ID)
				}
				err := database.Repos(db).Delete(ctx, ids...)
				if err != nil {
					t.Fatal(err)
				}
			})

			idMapping := make(map[api.RepoName]api.RepoID)
			for _, r := range stored {
				lower := strings.ToLower(string(r.Name))
				idMapping[api.RepoName(lower)] = r.ID
			}

			// Add gitserver_repos rows
			for _, toClone := range tc.gitserverCloned {
				toClone = strings.ToLower(toClone)
				id := idMapping[api.RepoName(toClone)]
				if id == 0 {
					continue
				}
				lastError := ""
				if tc.gitserverFailure != nil && tc.gitserverFailure[toClone] {
					lastError = "Oops"
				}
				if err := database.GitserverRepos(db).Upsert(ctx, &types.GitserverRepo{
					RepoID:      id,
					ShardID:     "test",
					CloneStatus: types.CloneStatusCloned,
					LastError:   lastError,
				}); err != nil {
					t.Fatal(err)
				}
			}
			t.Cleanup(func() {
				err = store.Exec(ctx, sqlf.Sprintf(`DELETE FROM gitserver_repos`))
				if err != nil {
					t.Fatal(err)
				}
			})

			// Set up ownership of repos
			if tc.repoOwner != nil {
				for _, repo := range stored {
					svc, ok := tc.repoOwner[repo.Name]
					if !ok {
						continue
					}
					err = store.Exec(ctx, sqlf.Sprintf(`
						INSERT INTO external_service_repos(external_service_id, repo_id, user_id, clone_url)
						VALUES (%s, %s, NULLIF(%s, 0), 'example.com')
					`, svc.ID, repo.ID, svc.NamespaceUserID))
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() {
						err = store.Exec(ctx, sqlf.Sprintf(`DELETE FROM external_service_repos WHERE external_service_id = %s`, svc.ID))
						if err != nil {
							t.Fatal(err)
						}
					})
				}
			}

			clock := timeutil.NewFakeClock(time.Now(), 0)
			syncer := &Syncer{
				Store: store,
				Now:   clock.Now,
			}

			if tc.sourcerErr != nil || tc.listRepoErr != nil {
				database.Mocks.Repos.List = func(v0 context.Context, v1 database.ReposListOptions) ([]*types.Repo, error) {
					return nil, tc.listRepoErr
				}
				defer func() {
					database.Mocks.Repos.List = nil
				}()
				sourcer := NewFakeSourcer(tc.sourcerErr, NewFakeSource(siteLevelService, nil))
				syncer.Sourcer = sourcer

				err = syncer.SyncExternalService(ctx, store, siteLevelService.ID, time.Millisecond)
				// In prod, SyncExternalService is kicked off by a worker queue. Any error
				// returned will be stored in the external_service_sync_jobs table so we fake
				// that here.
				if err != nil {
					defer func() { database.Mocks.ExternalServices = database.MockExternalServices{} }()
					database.Mocks.ExternalServices.ListSyncErrors = func(ctx context.Context) (map[int64]string, error) {
						return map[int64]string{
							siteLevelService.ID: err.Error(),
						}, nil
					}
				}
			}

			if tc.err == "" {
				tc.err = "<nil>"
			}

			res, err := FetchStatusMessages(ctx, db, tc.user)
			if have, want := fmt.Sprint(err), tc.err; have != want {
				t.Errorf("have err: %q, want: %q", have, want)
			}

			if diff := cmp.Diff(tc.res, res); diff != "" {
				t.Error(diff)
			}
		})
	}
}
