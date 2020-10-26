package repos_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/awscodecommit"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitolite"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/schema"
)

func testStoreListExternalServicesByRepos(t *testing.T, store repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		clock := repos.NewFakeClock(time.Now(), 0)
		now := clock.Now()

		t.Run("", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			github := repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			gitlab := repos.ExternalService{
				Kind:        extsvc.KindGitLab,
				DisplayName: "GitLab - Test",
				Config:      `{"url": "https://gitlab.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			svcs := repos.ExternalServices{&github, &gitlab}

			if err := tx.UpsertExternalServices(ctx, svcs...); err != nil {
				t.Fatalf("failed to setup store: %v", err)
			}

			repositories := repos.Repos{
				{
					Name: "github.com/foo/bar",
					Sources: map[string]*repos.SourceInfo{
						fmt.Sprintf("extsvc:github:%d", github.ID): {
							ID: fmt.Sprintf("extsvc:github:%d", github.ID),
						},
					},
					ExternalRepo: api.ExternalRepoSpec{
						ID:          "bar",
						ServiceType: "github",
						ServiceID:   "http://github.com",
					},
				},
				{
					Name: "github.com/foo/baz",
					Sources: map[string]*repos.SourceInfo{
						fmt.Sprintf("extsvc:github:%d", github.ID): {
							ID: fmt.Sprintf("extsvc:github:%d", github.ID),
						},
					},
					ExternalRepo: api.ExternalRepoSpec{
						ID:          "baz",
						ServiceType: "github",
						ServiceID:   "http://github.com",
					},
				},
				{
					Name: "gitlab.com/foo/bar",
					Sources: map[string]*repos.SourceInfo{
						fmt.Sprintf("extsvc:gitlab:%d", gitlab.ID): {
							ID: fmt.Sprintf("extsvc:gitlab:%d", gitlab.ID),
						},
					},
					ExternalRepo: api.ExternalRepoSpec{
						ID:          "bar",
						ServiceType: extsvc.TypeGitLab,
						ServiceID:   "http://gitlab.com",
					},
				},
			}

			if err := tx.InsertRepos(ctx, repositories...); err != nil {
				t.Fatalf("failed to setup store: %v", err)
			}

			opts := repos.StoreListExternalServicesArgs{
				RepoIDs: repositories.IDs(),
			}

			have, err := tx.ListExternalServices(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			repos.Assert.ExternalServicesEqual(svcs...)(t, have)
		}))
	}
}

func testStoreListExternalServices(userID int32) func(*testing.T, repos.Store) func(*testing.T) {
	return func(t *testing.T, store repos.Store) func(*testing.T) {
		clock := repos.NewFakeClock(time.Now(), 0)
		now := clock.Now()

		github := repos.ExternalService{
			Kind:            extsvc.KindGitHub,
			DisplayName:     "Github - Test",
			Config:          `{"url": "https://github.com"}`,
			NamespaceUserID: userID,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		gitlab := repos.ExternalService{
			Kind:        extsvc.KindGitLab,
			DisplayName: "GitLab - Test",
			Config:      `{"url": "https://gitlab.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		bitbucketServer := repos.ExternalService{
			Kind:        extsvc.KindBitbucketServer,
			DisplayName: "Bitbucket Server - Test",
			Config:      `{"url": "https://bitbucketserver.mycorp.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		awsCodeCommit := repos.ExternalService{
			Kind:        extsvc.KindAWSCodeCommit,
			DisplayName: "AWS CodeCommit - Test",
			Config:      `{"region": "us-west-1"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		otherService := repos.ExternalService{
			Kind:        extsvc.KindOther,
			DisplayName: "Other code hosts",
			Config:      `{"url": "https://git-host.mycorp.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		gitoliteService := repos.ExternalService{
			Kind:        extsvc.KindGitolite,
			DisplayName: "Gitolite Server - Test",
			Config:      `{"prefix": "/", "host": "git@gitolite.mycorp.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		phabricatorService := repos.ExternalService{
			Kind:        extsvc.KindPhabricator,
			DisplayName: "Phabricator - Test",
			Config:      `{"url": "https://phab.org", "token": "foo"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		svcs := repos.ExternalServices{
			&github,
			&gitlab,
			&bitbucketServer,
			&awsCodeCommit,
			&otherService,
			&gitoliteService,
			&phabricatorService,
		}

		type testCase struct {
			name string
			args func(stored repos.ExternalServices) repos.StoreListExternalServicesArgs

			stored repos.ExternalServices
			assert repos.ExternalServicesAssertion
			err    error
		}

		var testCases []testCase
		testCases = append(testCases,
			testCase{
				name: "returned kind is uppercase",
				args: func(repos.ExternalServices) repos.StoreListExternalServicesArgs {
					return repos.StoreListExternalServicesArgs{
						Kinds: svcs.Kinds(),
					}
				},
				stored: svcs,
				assert: repos.Assert.ExternalServicesEqual(svcs...),
			},
			testCase{
				name: "case-insensitive kinds",
				args: func(repos.ExternalServices) (args repos.StoreListExternalServicesArgs) {
					for _, kind := range svcs.Kinds() {
						args.Kinds = append(args.Kinds, strings.ToLower(kind))
					}
					return args
				},
				stored: svcs,
				assert: repos.Assert.ExternalServicesEqual(svcs...),
			},
			testCase{
				name:   "excludes soft deleted external services by default",
				stored: svcs.With(repos.Opt.ExternalServiceDeletedAt(now)),
				assert: repos.Assert.ExternalServicesEqual(),
			},
			testCase{
				name:   "results are in ascending order by id",
				stored: generateExternalServices(7, svcs...),
				assert: repos.Assert.ExternalServicesOrderedBy(
					func(a, b *repos.ExternalService) bool {
						return a.ID < b.ID
					},
				),
			},
			testCase{
				name:   "excludes phabricator by default",
				stored: svcs,
				assert: repos.Assert.ExternalServicesEqual(func() (es repos.ExternalServices) {
					for _, e := range svcs {
						if e.Kind != extsvc.KindPhabricator {
							es = append(es, e)
						}
					}
					return es
				}()...),
			},
			testCase{
				name:   "includes phabricator if specified in Kinds",
				stored: svcs,
				args: func(repos.ExternalServices) (args repos.StoreListExternalServicesArgs) {
					args.Kinds = []string{extsvc.KindPhabricator}
					return args
				},
				assert: repos.Assert.ExternalServicesEqual(&phabricatorService),
			},
			testCase{
				name:   "returns svcs by their ids",
				stored: svcs,
				args: func(stored repos.ExternalServices) repos.StoreListExternalServicesArgs {
					return repos.StoreListExternalServicesArgs{
						IDs: []int64{stored[0].ID, stored[1].ID},
					}
				},
				assert: repos.Assert.ExternalServicesEqual(svcs[:2].Clone()...),
			},
			testCase{
				name:   "filter services by owner",
				stored: svcs,
				args: func(stored repos.ExternalServices) repos.StoreListExternalServicesArgs {
					return repos.StoreListExternalServicesArgs{
						NamespaceUserID: userID,
					}
				},
				assert: repos.Assert.ExternalServicesEqual(svcs[:1].Clone()...),
			},
			testCase{
				name:   "fetch services with NO owner",
				stored: svcs,
				args: func(stored repos.ExternalServices) repos.StoreListExternalServicesArgs {
					return repos.StoreListExternalServicesArgs{
						NamespaceUserID: -1,
					}
				},
				// Skip GitHub since it has an owner
				// Also don't expect Phabricator since by default we should not include it
				assert: repos.Assert.ExternalServicesEqual(svcs[1:6].Clone()...),
			},
			testCase{
				name:   "limit and zero cursor",
				stored: svcs,
				args: func(repos.ExternalServices) (args repos.StoreListExternalServicesArgs) {
					args.Cursor = 0
					args.Limit = 1
					return args
				},
				assert: repos.Assert.ExternalServicesEqual(func() (es repos.ExternalServices) {
					return repos.ExternalServices{
						svcs[0],
					}
				}()...),
			},
			testCase{
				name:   "limit and non-zero cursor",
				stored: svcs,
				args: func(repos repos.ExternalServices) (args repos.StoreListExternalServicesArgs) {
					args.Cursor = repos[0].ID
					args.Limit = 1
					return args
				},
				assert: repos.Assert.ExternalServicesEqual(func() (es repos.ExternalServices) {
					return repos.ExternalServices{
						svcs[1],
					}
				}()...),
			},
		)

		return func(t *testing.T) {
			t.Helper()

			for _, tc := range testCases {
				tc := tc
				ctx := context.Background()

				t.Run(tc.name, transact(ctx, store, func(t testing.TB, tx repos.Store) {
					stored := tc.stored.Clone()
					if err := tx.UpsertExternalServices(ctx, stored...); err != nil {
						t.Fatalf("failed to setup store: %v", err)
					}

					var args repos.StoreListExternalServicesArgs
					if tc.args != nil {
						args = tc.args(stored)
					}

					es, err := tx.ListExternalServices(ctx, args)
					if have, want := fmt.Sprint(err), fmt.Sprint(tc.err); have != want {
						t.Errorf("error:\nhave: %v\nwant: %v", have, want)
					}

					for i, svc := range es {
						t.Logf("Service %d: %#v\n", i, svc)
					}
					if tc.assert != nil {
						tc.assert(t, es)
					}
				}))
			}
		}
	}
}

func testStoreUpsertExternalServices(t *testing.T, store repos.Store) func(*testing.T) {
	clock := repos.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	return func(t *testing.T) {
		github := repos.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		gitlab := repos.ExternalService{
			Kind:        extsvc.KindGitLab,
			DisplayName: "GitLab - Test",
			Config:      `{"url": "https://gitlab.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		bitbucketServer := repos.ExternalService{
			Kind:        extsvc.KindBitbucketServer,
			DisplayName: "Bitbucket Server - Test",
			Config:      `{"url": "https://bitbucketserver.mycorp.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		awsCodeCommit := repos.ExternalService{
			Kind:        extsvc.KindAWSCodeCommit,
			DisplayName: "AWS CodeCommit - Test",
			Config:      `{"region": "us-west-1"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		otherService := repos.ExternalService{
			Kind:        extsvc.KindOther,
			DisplayName: "Other code hosts",
			Config:      `{"url": "https://git-host.mycorp.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		gitoliteService := repos.ExternalService{
			Kind:        extsvc.KindGitolite,
			DisplayName: "Gitolite Server - Test",
			Config:      `{"prefix": "/", "host": "git@gitolite.mycorp.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		svcs := repos.ExternalServices{
			&github,
			&gitlab,
			&bitbucketServer,
			&awsCodeCommit,
			&otherService,
			&gitoliteService,
		}

		ctx := context.Background()

		t.Run("no external services", func(t *testing.T) {
			if err := store.UpsertExternalServices(ctx); err != nil {
				t.Fatalf("UpsertExternalServices error: %s", err)
			}
		})

		t.Run("many external services", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			want := generateExternalServices(7, svcs...)

			if err := tx.UpsertExternalServices(ctx, want...); err != nil {
				t.Fatalf("UpsertExternalServices error: %s", err)
			}

			for _, e := range want {
				if e.Kind != strings.ToUpper(e.Kind) {
					t.Errorf("external service kind didn't get upper-cased: %q", e.Kind)
					break
				}
			}

			sort.Sort(want)

			have, err := tx.ListExternalServices(ctx, repos.StoreListExternalServicesArgs{
				Kinds: svcs.Kinds(),
			})
			if err != nil {
				t.Fatalf("ListExternalServices error: %s", err)
			}

			if diff := cmp.Diff(have, []*repos.ExternalService(want), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListExternalServices:\n%s", diff)
			}

			now := clock.Now()
			suffix := "-updated"
			for _, r := range want {
				r.DisplayName += suffix
				r.Kind += suffix
				r.Config += suffix
				r.UpdatedAt = now
				r.CreatedAt = now
			}

			if err = tx.UpsertExternalServices(ctx, want...); err != nil {
				t.Errorf("UpsertExternalServices error: %s", err)
			} else if have, err = tx.ListExternalServices(ctx, repos.StoreListExternalServicesArgs{}); err != nil {
				t.Errorf("ListExternalServices error: %s", err)
			} else if diff := cmp.Diff(have, []*repos.ExternalService(want), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListExternalServices:\n%s", diff)
			}

			want.Apply(repos.Opt.ExternalServiceDeletedAt(now))
			args := repos.StoreListExternalServicesArgs{}

			if err = tx.UpsertExternalServices(ctx, want.Clone()...); err != nil {
				t.Errorf("UpsertExternalServices error: %s", err)
			} else if have, err = tx.ListExternalServices(ctx, args); err != nil {
				t.Errorf("ListExternalServices error: %s", err)
			} else if diff := cmp.Diff(have, []*repos.ExternalService(nil), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListExternalServices:\n%s", diff)
			}
		}))
	}
}

func testStoreInsertRepos(t *testing.T, store repos.Store) func(*testing.T) {
	clock := repos.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	return func(t *testing.T) {
		servicesPerKind := createExternalServices(t, store)

		repo1 := repos.Repo{
			Name:        "github.com/foo/bar",
			URI:         "github.com/foo/bar",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "AAAAA==",
				ServiceType: "github",
				ServiceID:   "http://github.com",
			},
			Sources: map[string]*repos.SourceInfo{
				servicesPerKind[extsvc.KindGitHub].URN(): {
					ID:       servicesPerKind[extsvc.KindGitHub].URN(),
					CloneURL: "git@github.com:foo/bar.git",
				},
				servicesPerKind[extsvc.KindBitbucketServer].URN(): {
					ID:       servicesPerKind[extsvc.KindBitbucketServer].URN(),
					CloneURL: "git@bitbucketserver.mycorp.com:foo/bar.git",
				},
			},
			Metadata: new(github.Repository),
		}

		repo2 := repos.Repo{
			Name:        "gitlab.com/foo/bar",
			URI:         "gitlab.com/foo/bar",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "1234",
				ServiceType: extsvc.TypeGitLab,
				ServiceID:   "http://gitlab.com",
			},
			Sources: map[string]*repos.SourceInfo{
				servicesPerKind[extsvc.KindGitLab].URN(): {
					ID:       servicesPerKind[extsvc.KindGitLab].URN(),
					CloneURL: "git@gitlab.com:foo/bar.git",
				},
			},
			Metadata: new(gitlab.Project),
		}

		ctx := context.Background()

		t.Run("no repos should not fail", func(t *testing.T) {
			if err := store.InsertRepos(ctx); err != nil {
				t.Fatalf("InsertRepos error: %s", err)
			}
		})

		t.Run("many repos", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			want := mkRepos(7, &repo1, &repo2)

			if err := tx.InsertRepos(ctx, want...); err != nil {
				t.Fatalf("InsertRepos error: %s", err)
			}

			sort.Sort(want)

			if noID := want.Filter(hasNoID); len(noID) > 0 {
				t.Fatalf("InsertRepos didn't assign an ID to all repos: %v", noID.Names())
			}

			have, err := tx.ListRepos(ctx, repos.StoreListReposArgs{})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff(have, []*repos.Repo(want), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}
		}))
	}
}

func testStoreDeleteRepos(t *testing.T, store repos.Store) func(*testing.T) {
	clock := repos.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	return func(t *testing.T) {
		servicesPerKind := createExternalServices(t, store)

		repo1 := repos.Repo{
			Name:        "github.com/foo/bar",
			URI:         "github.com/foo/bar",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "AAAAA==",
				ServiceType: "github",
				ServiceID:   "http://github.com",
			},
			Sources: map[string]*repos.SourceInfo{
				servicesPerKind[extsvc.KindGitHub].URN(): {
					ID:       servicesPerKind[extsvc.KindGitHub].URN(),
					CloneURL: "git@github.com:foo/bar.git",
				},
				servicesPerKind[extsvc.KindBitbucketServer].URN(): {
					ID:       servicesPerKind[extsvc.KindBitbucketServer].URN(),
					CloneURL: "git@bitbucketserver.mycorp.com:foo/bar.git",
				},
			},
			Metadata: new(github.Repository),
		}

		repo2 := repos.Repo{
			Name:        "gitlab.com/foo/bar",
			URI:         "gitlab.com/foo/bar",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "1234",
				ServiceType: extsvc.TypeGitLab,
				ServiceID:   "http://gitlab.com",
			},
			Sources: map[string]*repos.SourceInfo{
				servicesPerKind[extsvc.KindGitLab].URN(): {
					ID:       servicesPerKind[extsvc.KindGitLab].URN(),
					CloneURL: "git@gitlab.com:foo/bar.git",
				},
			},
			Metadata: new(gitlab.Project),
		}

		ctx := context.Background()

		t.Run("no repos should not fail", func(t *testing.T) {
			if err := store.DeleteRepos(ctx); err != nil {
				t.Fatalf("DeleteRepos error: %s", err)
			}
		})

		t.Run("many repos", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			rs := mkRepos(7, &repo1, &repo2)

			if err := tx.InsertRepos(ctx, rs...); err != nil {
				t.Fatalf("InsertRepos error: %s", err)
			}

			sort.Sort(rs)

			toDelete, want := rs[:3], rs[3:]

			if err := tx.DeleteRepos(ctx, toDelete.IDs()...); err != nil {
				t.Fatalf("DeleteRepos error: %s", err)
			}

			have, err := tx.ListRepos(ctx, repos.StoreListReposArgs{})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff(have, []*repos.Repo(want), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}
		}))
	}
}

func testStoreUpsertRepos(t *testing.T, store repos.Store) func(*testing.T) {
	clock := repos.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	return func(t *testing.T) {
		kinds := []string{
			extsvc.KindGitHub,
			extsvc.KindGitLab,
			extsvc.KindBitbucketServer,
			extsvc.KindAWSCodeCommit,
			extsvc.KindOther,
			extsvc.KindGitolite,
		}

		servicesPerKind := createExternalServices(t, store)

		github := repos.Repo{
			Name:        "github.com/foo/bar",
			URI:         "github.com/foo/bar",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "AAAAA==",
				ServiceType: "github",
				ServiceID:   "http://github.com",
			},
			Sources: map[string]*repos.SourceInfo{
				servicesPerKind[extsvc.KindGitHub].URN(): {
					ID:       servicesPerKind[extsvc.KindGitHub].URN(),
					CloneURL: "git@github.com:foo/bar.git",
				},
			},
			Metadata: new(github.Repository),
		}

		gitlab := repos.Repo{
			Name:        "gitlab.com/foo/bar",
			URI:         "gitlab.com/foo/bar",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "1234",
				ServiceType: extsvc.TypeGitLab,
				ServiceID:   "http://gitlab.com",
			},
			Sources: map[string]*repos.SourceInfo{
				servicesPerKind[extsvc.KindGitLab].URN(): {
					ID:       servicesPerKind[extsvc.KindGitLab].URN(),
					CloneURL: "git@gitlab.com:foo/bar.git",
				},
			},
			Metadata: new(gitlab.Project),
		}

		bitbucketServer := repos.Repo{
			Name:        "bitbucketserver.mycorp.com/foo/bar",
			URI:         "bitbucketserver.mycorp.com/foo/bar",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "1234",
				ServiceType: "bitbucketServer",
				ServiceID:   "http://bitbucketserver.mycorp.com",
			},
			Sources: map[string]*repos.SourceInfo{
				servicesPerKind[extsvc.KindBitbucketServer].URN(): {
					ID:       servicesPerKind[extsvc.KindBitbucketServer].URN(),
					CloneURL: "git@bitbucketserver.mycorp.com:foo/bar.git",
				},
			},
			Metadata: new(bitbucketserver.Repo),
		}

		awsCodeCommit := repos.Repo{
			Name:        "git-codecommit.us-west-1.amazonaws.com/stripe-go",
			URI:         "git-codecommit.us-west-1.amazonaws.com/stripe-go",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "f001337a-3450-46fd-b7d2-650c0EXAMPLE",
				ServiceType: extsvc.TypeAWSCodeCommit,
				ServiceID:   "arn:aws:codecommit:us-west-1:999999999999:",
			},
			Sources: map[string]*repos.SourceInfo{
				servicesPerKind[extsvc.KindAWSCodeCommit].URN(): {
					ID:       servicesPerKind[extsvc.KindAWSCodeCommit].URN(),
					CloneURL: "git@git-codecommit.us-west-1.amazonaws.com/v1/repos/stripe-go",
				},
			},
			Metadata: new(awscodecommit.Repository),
		}

		otherRepo := repos.Repo{
			Name: "git-host.com/org/foo",
			URI:  "git-host.com/org/foo",
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "git-host.com/org/foo",
				ServiceID:   "https://git-host.com/",
				ServiceType: extsvc.TypeOther,
			},
			Sources: map[string]*repos.SourceInfo{
				servicesPerKind[extsvc.KindOther].URN(): {
					ID:       servicesPerKind[extsvc.KindOther].URN(),
					CloneURL: "https://git-host.com/org/foo",
				},
			},
		}

		gitoliteRepo := repos.Repo{
			Name:      "gitolite.mycorp.com/bar",
			URI:       "gitolite.mycorp.com/bar",
			CreatedAt: now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "bar",
				ServiceType: extsvc.TypeGitolite,
				ServiceID:   "git@gitolite.mycorp.com",
			},
			Sources: map[string]*repos.SourceInfo{
				servicesPerKind[extsvc.KindGitolite].URN(): {
					ID:       servicesPerKind[extsvc.KindGitolite].URN(),
					CloneURL: "git@gitolite.mycorp.com:bar.git",
				},
			},
			Metadata: new(gitolite.Repo),
		}

		repositories := repos.Repos{
			&github,
			&gitlab,
			&bitbucketServer,
			&awsCodeCommit,
			&otherRepo,
			&gitoliteRepo,
		}

		ctx := context.Background()

		t.Run("no repos", func(t *testing.T) {
			if err := store.UpsertRepos(ctx); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}
		})

		t.Run("many repos", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			want := mkRepos(7, repositories...)

			if err := tx.UpsertRepos(ctx, want...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}
			if err := tx.UpsertSources(ctx, want.Sources(), nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			sort.Sort(want)

			if noID := want.Filter(hasNoID); len(noID) > 0 {
				t.Fatalf("UpsertRepos didn't assign an ID to all repos: %v", noID.Names())
			}

			have, err := tx.ListRepos(ctx, repos.StoreListReposArgs{
				Kinds: kinds,
			})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff(have, []*repos.Repo(want), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}

			suffix := "-updated"
			now := clock.Now()
			for _, r := range want {
				r.Name += suffix
				r.URI += suffix
				r.Description += suffix
				r.UpdatedAt = now
				r.CreatedAt = now
				r.Archived = !r.Archived
				r.Fork = !r.Fork
			}

			if err = tx.UpsertRepos(ctx, want.Clone()...); err != nil {
				t.Errorf("UpsertRepos error: %s", err)
			} else if err = tx.UpsertSources(ctx, want.Clone().Sources(), nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			} else if have, err = tx.ListRepos(ctx, repos.StoreListReposArgs{}); err != nil {
				t.Errorf("ListRepos error: %s", err)
			} else if diff := cmp.Diff(have, []*repos.Repo(want), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListRepos:\n%s", diff)
			}

			deleted := want.Clone().With(repos.Opt.RepoDeletedAt(now))
			args := repos.StoreListReposArgs{}

			if err = tx.UpsertRepos(ctx, deleted...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			} else if have, err = tx.ListRepos(ctx, args); err != nil {
				t.Errorf("ListRepos error: %s", err)
			} else if diff := cmp.Diff(have, []*repos.Repo(nil), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListRepos:\n%s", diff)
			}

			// Insert previously soft-deleted repos. Ensure we get back the same ID.
			if err = tx.UpsertRepos(ctx, want.Clone().With(repos.Opt.RepoID(0))...); err != nil {
				t.Errorf("UpsertRepos error: %s", err)
			} else if err = tx.UpsertSources(ctx, want.Clone().Sources(), nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			} else if have, err = tx.ListRepos(ctx, repos.StoreListReposArgs{}); err != nil {
				t.Errorf("ListRepos error: %s", err)
			} else if diff := cmp.Diff(have, []*repos.Repo(want), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListRepos:\n%s", diff)
			}

			// Delete all again, then try insert repos with different external
			// IDs but same name. Check we get new IDs.
			for _, r := range want {
				r.ID = 0
				r.ExternalRepo.ID += "-different"
			}
			if err = tx.UpsertRepos(ctx, deleted...); err != nil {
				t.Fatalf("UpsertRepos deleted error: %s", err)
			} else if err = tx.UpsertRepos(ctx, want...); err != nil {
				t.Fatalf("UpsertRepos want error: %s", err)
			} else if err = tx.UpsertSources(ctx, want.Sources(), nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			} else if have, err = tx.ListRepos(ctx, repos.StoreListReposArgs{}); err != nil {
				t.Errorf("ListRepos error: %s", err)
			} else if diff := cmp.Diff(have, []*repos.Repo(want), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListRepos:\n%s", diff)
			} else if sameIDs := want.Filter(hasID(deleted.IDs()...)); len(sameIDs) > 0 {
				t.Errorf("ListRepos returned IDs of soft deleted repos: %v", sameIDs.Names())
			}
		}))

		t.Run("many repos soft-deleted and single repo reinserted", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			all := mkRepos(7, repositories...)

			if err := tx.UpsertRepos(ctx, all...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}
			if err := tx.UpsertSources(ctx, all.Sources(), nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			sort.Sort(all)

			if noID := all.Filter(hasNoID); len(noID) > 0 {
				t.Fatalf("UpsertRepos didn't assign an ID to all repos: %v", noID.Names())
			}

			have, err := tx.ListRepos(ctx, repos.StoreListReposArgs{
				Kinds: kinds,
			})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff(have, []*repos.Repo(all), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}

			allDeleted := all.Clone().With(repos.Opt.RepoDeletedAt(now))
			args := repos.StoreListReposArgs{}

			if err = tx.UpsertRepos(ctx, allDeleted...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			} else if have, err = tx.ListRepos(ctx, args); err != nil {
				t.Errorf("ListRepos error: %s", err)
			} else if diff := cmp.Diff(have, []*repos.Repo(nil), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListRepos:\n%s", diff)
			}

			// Insert one of the previously soft-deleted repos. Ensure ID on upserted repo is set and we get back the same ID.
			want := repos.Repos{all[0]}
			upsert := want.Clone().With(repos.Opt.RepoID(0))
			if err = tx.UpsertRepos(ctx, upsert...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}
			if upsert[0].ID == 0 {
				t.Fatalf("Repo ID is zero")
			}
			if err := tx.UpsertSources(ctx, upsert.Sources(), nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			if have, err = tx.ListRepos(ctx, repos.StoreListReposArgs{}); err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}
			if diff := cmp.Diff(have, []*repos.Repo(want), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}
		}))

		t.Run("it shouldn't modify the cloned column", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			// UpsertRepos shouldn't set the cloned column to true
			r := mkRepos(1, repositories...)[0]
			r.Cloned = true
			if err := tx.UpsertRepos(ctx, r); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			count, err := tx.CountNotClonedRepos(ctx)
			if err != nil {
				t.Fatalf("CountNotClonedRepos error: %s", err)
			}
			if count != 1 {
				t.Fatalf("Wrong number of not cloned repos: %d", count)
			}

			// UpsertRepos shouldn't set the cloned column to false either
			if err := tx.SetClonedRepos(ctx, r.Name); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}
			r = r.Clone()
			r.Cloned = false
			if err := tx.UpsertRepos(ctx, r); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			count, err = tx.CountNotClonedRepos(ctx)
			if err != nil {
				t.Fatalf("CountNotClonedRepos error: %s", err)
			}
			if count != 0 {
				t.Fatalf("Wrong number of not cloned repos: %d", count)
			}
		}))
	}
}

func testStoreUpsertSources(t *testing.T, store repos.Store) func(*testing.T) {
	clock := repos.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	servicesPerKind := createExternalServices(t, store)

	return func(t *testing.T) {
		github := repos.Repo{
			Name:        "github.com/foo/bar",
			URI:         "github.com/foo/bar",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "AAAAA==",
				ServiceType: "github",
				ServiceID:   "http://github.com",
			},
			Sources: map[string]*repos.SourceInfo{
				servicesPerKind[extsvc.KindGitHub].URN(): {
					ID:       servicesPerKind[extsvc.KindGitHub].URN(),
					CloneURL: "git@github.com:foo/bar.git",
				},
			},
			Metadata: new(github.Repository),
		}

		gitlab := repos.Repo{
			Name:        "gitlab.com/foo/bar",
			URI:         "gitlab.com/foo/bar",
			Description: "The description",
			CreatedAt:   now,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "1234",
				ServiceType: extsvc.TypeGitLab,
				ServiceID:   "http://gitlab.com",
			},
			Sources: map[string]*repos.SourceInfo{
				servicesPerKind[extsvc.KindGitLab].URN(): {
					ID:       servicesPerKind[extsvc.KindGitLab].URN(),
					CloneURL: "git@gitlab.com:foo/bar.git",
				},
			},
			Metadata: new(gitlab.Project),
		}

		repositories := repos.Repos{
			&github,
			&gitlab,
		}

		ctx := context.Background()

		t.Run("no sources", func(t *testing.T) {
			if err := store.UpsertSources(ctx, nil, nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}
		})

		t.Run("delete repo", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			want := mkRepos(7, repositories...)

			if err := tx.UpsertRepos(ctx, want...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			sources := want.Sources()

			if err := tx.UpsertSources(ctx, sources, nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			// delete a repository
			want[0].DeletedAt = now
			if err := tx.DeleteRepos(ctx, want[0].ID); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			// un delete it
			want[0].DeletedAt = time.Time{}
			if err := tx.UpsertRepos(ctx, want...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			// it should not contain any source
			want[0].Sources = nil

			got, err := tx.ListRepos(ctx, repos.StoreListReposArgs{})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff([]*repos.Repo(want), got, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}
		}))

		t.Run("delete external service", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			origRepos := mkRepos(7, repositories...)

			if err := tx.UpsertRepos(ctx, origRepos...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			sources := origRepos.Sources()

			if err := tx.UpsertSources(ctx, sources, nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			// delete an external service
			svc := servicesPerKind[extsvc.KindGitHub]
			svc.DeletedAt = now
			if err := tx.UpsertExternalServices(ctx, svc); err != nil {
				t.Fatalf("UpsertExternalServices error: %s", err)
			}

			// un delete it
			svc.DeletedAt = time.Time{}
			if err := tx.UpsertExternalServices(ctx, svc); err != nil {
				t.Fatalf("UpsertExternalServices error: %s", err)
			}

			// All GitHub sources should be deleted and all orphan repositories should be excluded
			want := make([]*repos.Repo, 0, len(origRepos))
			origRepos.Apply(func(r *repos.Repo) {
				for urn := range r.Sources {
					if strings.Contains(urn, "github") {
						delete(r.Sources, urn)
					}
				}
				if len(r.Sources) > 0 {
					want = append(want, r)
				}
			})

			got, err := tx.ListRepos(ctx, repos.StoreListReposArgs{})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}
		}))

		t.Run("inserts updates and deletes", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			want := mkRepos(7, repositories...)

			if err := tx.UpsertRepos(ctx, want...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			sources := want.Sources()

			if err := tx.UpsertSources(ctx, sources, nil, nil); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			have, err := tx.ListRepos(ctx, repos.StoreListReposArgs{})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff([]*repos.Repo(want), have, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}

			updates := make(map[api.RepoID][]repos.SourceInfo)
			deletes := make(map[api.RepoID][]repos.SourceInfo)

			updates[want[0].ID] = sources[want[0].ID]
			updates[want[0].ID][0].CloneURL = "something-else"
			deletes[want[1].ID] = sources[want[1].ID]

			if err := tx.UpsertSources(ctx, nil, updates, deletes); err != nil {
				t.Fatalf("UpsertSources error: %s", err)
			}

			want[0].Sources[servicesPerKind[extsvc.KindGitHub].URN()] = &repos.SourceInfo{
				CloneURL: "something-else",
				ID:       servicesPerKind[extsvc.KindGitHub].URN(),
			}

			// Remove the second element from want because it should be deleted automatically
			// by the time it become orphaned.
			want = append(want[:1], want[2:]...)

			have, err = tx.ListRepos(ctx, repos.StoreListReposArgs{})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			if diff := cmp.Diff([]*repos.Repo(want), have, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListRepos:\n%s", diff)
			}
		}))
	}
}

func isCloned(r *repos.Repo) bool {
	return r.Cloned
}

func testStoreSetClonedRepos(t *testing.T, store repos.Store) func(*testing.T) {
	servicesPerKind := createExternalServices(t, store)

	return func(t *testing.T) {
		var repositories repos.Repos
		for i := 0; i < 3; i++ {
			repositories = append(repositories, &repos.Repo{
				Name:   fmt.Sprintf("github.com/%d/%d", i, i),
				URI:    fmt.Sprintf("github.com/%d/%d", i, i),
				Cloned: false,
				ExternalRepo: api.ExternalRepoSpec{
					ID:          strconv.Itoa(i),
					ServiceType: extsvc.TypeGitHub,
					ServiceID:   "http://github.com",
				},
				Sources: map[string]*repos.SourceInfo{
					servicesPerKind[extsvc.KindGitHub].URN(): {
						ID:       servicesPerKind[extsvc.KindGitHub].URN(),
						CloneURL: "git@github.com:foo/bar.git",
					},
				},
				Metadata: new(github.Repository),
			})
		}

		check := func(t testing.TB, ctx context.Context, tx repos.Store, wantNames []string) {
			t.Helper()

			res, err := tx.ListRepos(ctx, repos.StoreListReposArgs{})
			if err != nil {
				t.Fatalf("ListRepos error: %s", err)
			}

			cloned := repos.Repos(res).Filter(isCloned).Names()
			sort.Strings(cloned)

			if got, want := cloned, wantNames; !cmp.Equal(got, want) {
				t.Fatalf("got=%v, want=%v: %s", got, want, cmp.Diff(got, want))
			}
		}

		ctx := context.Background()

		t.Run("no repo name", func(t *testing.T) {
			if err := store.SetClonedRepos(ctx); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}
		})

		t.Run("many repo names", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			stored := mkRepos(9, repositories...)

			if err := tx.InsertRepos(ctx, stored...); err != nil {
				t.Fatalf("InsertRepos error: %s", err)
			}

			sort.Sort(stored)

			names := stored[:3].Names()
			sort.Strings(names)

			if err := tx.SetClonedRepos(ctx, names...); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}
			check(t, ctx, tx, names)

			// setClonedRepositories should be idempotent and have the same behavior
			// when called with the same repos
			if err := tx.SetClonedRepos(ctx, names...); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}
			check(t, ctx, tx, names)

			// when adding another repo to the list, the other repos must be set as well
			names = stored[:4].Names()
			sort.Strings(names)
			if err := tx.SetClonedRepos(ctx, names...); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}

			check(t, ctx, tx, names)
		}))

		t.Run("repo names in mixed case", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			stored := mkRepos(9, repositories...)
			for i := range stored {
				if i%2 == 0 {
					stored[i].Name = strings.ToUpper(stored[i].Name)
				}
			}

			if err := tx.InsertRepos(ctx, stored...); err != nil {
				t.Fatalf("InsertRepos error: %s", err)
			}

			sort.Sort(stored)

			originalNames := stored.Names()
			sort.Strings(originalNames)

			lowerCaseNames := make([]string, 0, len(originalNames))
			for _, n := range originalNames {
				lowerCaseNames = append(lowerCaseNames, strings.ToLower(n))
			}

			if err := tx.SetClonedRepos(ctx, lowerCaseNames...); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}

			check(t, ctx, tx, originalNames)
		}))
	}
}

func testStoreCountNotClonedRepos(t *testing.T, store repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		servicesPerKind := createExternalServices(t, store)

		var repositories repos.Repos
		for i := 0; i < 3; i++ {
			repositories = append(repositories, &repos.Repo{
				Name:   fmt.Sprintf("github.com/%d/%d", i, i),
				URI:    fmt.Sprintf("github.com/%d/%d", i, i),
				Cloned: false,
				ExternalRepo: api.ExternalRepoSpec{
					ID:          strconv.Itoa(i),
					ServiceType: extsvc.TypeGitHub,
					ServiceID:   "http://github.com",
				},
				Sources: map[string]*repos.SourceInfo{
					servicesPerKind[extsvc.KindGitHub].URN(): {
						ID:       servicesPerKind[extsvc.KindGitHub].URN(),
						CloneURL: "git@github.com:foo/bar.git",
					},
				},
				Metadata: new(github.Repository),
			})
		}

		ctx := context.Background()

		t.Run("no cloned repos", func(t *testing.T) {
			count, err := store.CountNotClonedRepos(ctx)
			if err != nil {
				t.Fatalf("CountNotClonedRepos error: %s", err)
			}
			if diff := cmp.Diff(count, uint64(0)); diff != "" {
				t.Fatalf("CountNotClonedRepos:\n%s", diff)
			}
		})

		t.Run("multiple cloned repos", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			stored := mkRepos(10, repositories...)

			if err := tx.InsertRepos(ctx, stored...); err != nil {
				t.Fatalf("InsertRepos error: %s", err)
			}

			sort.Sort(stored)
			cloned := stored[:3].Names()

			if err := tx.SetClonedRepos(ctx, cloned...); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}

			sort.Strings(cloned)

			count, err := tx.CountNotClonedRepos(ctx)
			if err != nil {
				t.Fatalf("CountNotClonedRepos error: %s", err)
			}
			if diff := cmp.Diff(count, uint64(7)); diff != "" {
				t.Fatalf("CountNotClonedRepos:\n%s", diff)
			}
		}))

		t.Run("deleted non cloned repos", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			stored := mkRepos(10, repositories...)

			if err := tx.InsertRepos(ctx, stored...); err != nil {
				t.Fatalf("InsertRepos error: %s", err)
			}

			sort.Sort(stored)
			cloned := stored[:3].Names()

			if err := tx.SetClonedRepos(ctx, cloned...); err != nil {
				t.Fatalf("SetClonedRepos error: %s", err)
			}

			sort.Strings(cloned)
			deletedCloned := stored[8:]

			if err := tx.DeleteRepos(ctx, deletedCloned.IDs()...); err != nil {
				t.Fatalf("UpsertRepos error: %s", err)
			}

			count, err := tx.CountNotClonedRepos(ctx)
			if err != nil {
				t.Fatalf("CountNotClonedRepos error: %s", err)
			}
			if diff := cmp.Diff(count, uint64(5)); diff != "" {
				t.Fatalf("CountNotClonedRepos:\n%s", diff)
			}
		}))
	}
}

func hasNoID(r *repos.Repo) bool {
	return r.ID == 0
}

func hasID(ids ...api.RepoID) func(r *repos.Repo) bool {
	return func(r *repos.Repo) bool {
		for _, id := range ids {
			if r.ID == id {
				return true
			}
		}
		return false
	}
}

func testStoreListRepos(t *testing.T, store repos.Store) func(*testing.T) {
	clock := repos.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	servicesPerKind := createExternalServices(t, store)

	unmanaged := repos.Repo{
		Name:     "unmanaged",
		Sources:  map[string]*repos.SourceInfo{},
		Metadata: new(github.Repository),
		ExternalRepo: api.ExternalRepoSpec{
			ServiceType: "non_existent_kind",
			ServiceID:   "https://example.com/",
			ID:          "unmanaged",
		},
	}

	github := repos.Repo{
		Name: "github.com/bar/foo",
		Sources: map[string]*repos.SourceInfo{
			servicesPerKind[extsvc.KindGitHub].URN(): {
				ID:       servicesPerKind[extsvc.KindGitHub].URN(),
				CloneURL: "git@github.com:bar/foo.git",
			},
		},
		Metadata: new(github.Repository),
		ExternalRepo: api.ExternalRepoSpec{
			ServiceType: "github",
			ServiceID:   "https://github.com/",
			ID:          "foo",
		},
	}

	gitlab := repos.Repo{
		Name:    "gitlab.com/bar/foo",
		Private: true,
		Sources: map[string]*repos.SourceInfo{
			servicesPerKind[extsvc.KindGitLab].URN(): {
				ID:       servicesPerKind[extsvc.KindGitLab].URN(),
				CloneURL: "git@gitlab.com:bar/foo.git",
			},
		},
		Metadata: new(gitlab.Project),
		ExternalRepo: api.ExternalRepoSpec{
			ServiceType: extsvc.TypeGitLab,
			ServiceID:   "https://gitlab.com/",
			ID:          "123",
		},
	}

	bitbucketServer := repos.Repo{
		Name: "bitbucketserver.mycorp.com/foo/bar",
		Sources: map[string]*repos.SourceInfo{
			servicesPerKind[extsvc.KindBitbucketServer].URN(): {
				ID:       servicesPerKind[extsvc.KindBitbucketServer].URN(),
				CloneURL: "git@bitbucketserver.mycorp.com:foo/bar.git",
			},
		},
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "1234",
			ServiceType: "bitbucketServer",
			ServiceID:   "http://bitbucketserver.mycorp.com",
		},
		Metadata: new(bitbucketserver.Repo),
	}

	awsCodeCommit := repos.Repo{
		Name: "git-codecommit.us-west-1.amazonaws.com/stripe-go",
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "f001337a-3450-46fd-b7d2-650c0EXAMPLE",
			ServiceType: extsvc.TypeAWSCodeCommit,
			ServiceID:   "arn:aws:codecommit:us-west-1:999999999999:",
		},
		Sources: map[string]*repos.SourceInfo{
			servicesPerKind[extsvc.KindAWSCodeCommit].URN(): {
				ID:       servicesPerKind[extsvc.KindAWSCodeCommit].URN(),
				CloneURL: "git@git-codecommit.us-west-1.amazonaws.com/v1/repos/stripe-go",
			},
		},
		Metadata: new(awscodecommit.Repository),
	}

	otherRepo := repos.Repo{
		Name: "git-host.com/org/foo",
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "git-host.com/org/foo",
			ServiceID:   "https://git-host.com/",
			ServiceType: extsvc.TypeOther,
		},
		Sources: map[string]*repos.SourceInfo{
			servicesPerKind[extsvc.KindOther].URN(): {
				ID:       servicesPerKind[extsvc.KindOther].URN(),
				CloneURL: "https://git-host.com/org/foo",
			},
		},
	}

	gitoliteRepo := repos.Repo{
		Name:      "gitolite.mycorp.com/bar",
		CreatedAt: now,
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "bar",
			ServiceType: extsvc.TypeGitolite,
			ServiceID:   "git@gitolite.mycorp.com",
		},
		Sources: map[string]*repos.SourceInfo{
			servicesPerKind[extsvc.KindGitolite].URN(): {
				ID:       servicesPerKind[extsvc.KindGitolite].URN(),
				CloneURL: "git@gitolite.mycorp.com:bar.git",
			},
		},
		Metadata: new(gitolite.Repo),
	}

	repositories := repos.Repos{
		&github,
		&gitlab,
		&bitbucketServer,
		&awsCodeCommit,
		&otherRepo,
		&gitoliteRepo,
	}

	kinds := []string{
		extsvc.KindGitHub,
		extsvc.KindGitLab,
		extsvc.KindBitbucketServer,
		extsvc.KindAWSCodeCommit,
		extsvc.KindOther,
		extsvc.KindGitolite,
	}

	type testCase struct {
		name   string
		args   func(stored repos.Repos) repos.StoreListReposArgs
		stored repos.Repos
		repos  repos.ReposAssertion
		err    error
	}

	var testCases []testCase
	{
		stored := repositories.With(func(r *repos.Repo) {
			r.ExternalRepo.ServiceType =
				strings.ToUpper(r.ExternalRepo.ServiceType)
		})

		testCases = append(testCases, testCase{
			name: "case-insensitive kinds",
			args: func(_ repos.Repos) (args repos.StoreListReposArgs) {
				for _, kind := range kinds {
					args.Kinds = append(args.Kinds, strings.ToUpper(kind))
				}
				return args
			},
			stored: stored,
			repos:  repos.Assert.ReposEqual(stored...),
		})
	}

	testCases = append(testCases, testCase{
		name: "ignores unmanaged",
		args: func(_ repos.Repos) repos.StoreListReposArgs {
			return repos.StoreListReposArgs{Kinds: kinds}
		},
		stored: repos.Repos{&github, &gitlab, &unmanaged}.Clone(),
		repos:  repos.Assert.ReposEqual(&github, &gitlab),
	})

	{
		stored := repositories.With(repos.Opt.RepoDeletedAt(now))
		testCases = append(testCases, testCase{
			name:   "excludes soft deleted repos by default",
			stored: stored,
			repos:  repos.Assert.ReposEqual(),
		})
	}

	testCases = append(testCases, testCase{
		name:   "returns repos in ascending order by id",
		stored: mkRepos(7, repositories...),
		repos: repos.Assert.ReposOrderedBy(func(a, b *repos.Repo) bool {
			return a.ID < b.ID
		}),
	})

	testCases = append(testCases, testCase{
		name:   "returns repos by their names",
		stored: repositories,
		args: func(_ repos.Repos) repos.StoreListReposArgs {
			return repos.StoreListReposArgs{
				Names: []string{github.Name, gitlab.Name},
			}
		},
		repos: repos.Assert.ReposEqual(&github, &gitlab),
	})

	testCases = append(testCases, testCase{
		name:   "returns repos by their ids",
		stored: repositories,
		args: func(stored repos.Repos) repos.StoreListReposArgs {
			return repos.StoreListReposArgs{
				IDs: []api.RepoID{stored[0].ID, stored[1].ID},
			}
		},
		repos: repos.Assert.ReposEqual(repositories[:2].Clone()...),
	})

	testCases = append(testCases, testCase{
		name:   "limits repos to the given kinds",
		stored: repositories,
		args: func(repos.Repos) repos.StoreListReposArgs {
			return repos.StoreListReposArgs{
				Kinds: []string{extsvc.KindGitHub, extsvc.KindGitLab},
			}
		},
		repos: repos.Assert.ReposEqual(&github, &gitlab),
	})

	testCases = append(testCases, testCase{
		name: "only include private",
		args: func(repos.Repos) repos.StoreListReposArgs {
			return repos.StoreListReposArgs{
				PrivateOnly: true,
			}
		},
		stored: repositories,
		repos:  repos.Assert.ReposEqual(&gitlab),
	})

	testCases = append(testCases, testCase{
		name:   "use or",
		stored: repositories,
		args: func(repos.Repos) repos.StoreListReposArgs {
			return repos.StoreListReposArgs{
				Names: []string{"gitlab.com/bar/foo"},
				Kinds: []string{"github"},
				UseOr: true,
			}
		},
		repos: repos.Assert.ReposEqual(&github, &gitlab),
	})

	testCases = append(testCases, testCase{
		name:   "use and",
		stored: repositories,
		args: func(repos.Repos) repos.StoreListReposArgs {
			return repos.StoreListReposArgs{
				Names: []string{"gitlab.com/bar/foo"},
				Kinds: []string{"github"},
				UseOr: false,
			}
		},
		repos: repos.Assert.ReposEqual(),
	})

	{
		testCases = append(testCases, testCase{
			name:   "limit by external service",
			stored: repositories,
			args: func(repos.Repos) repos.StoreListReposArgs {
				return repos.StoreListReposArgs{
					ExternalServiceID: servicesPerKind[extsvc.KindGitHub].ID,
				}
			},
			repos: repos.Assert.ReposEqual(&github),
		})
	}

	return func(t *testing.T) {
		ctx := context.Background()

		for _, tc := range testCases {
			tc := tc

			t.Run(tc.name, transact(ctx, store, func(t testing.TB, tx repos.Store) {
				stored := tc.stored.Clone()

				if err := tx.InsertRepos(ctx, stored...); err != nil {
					t.Fatalf("failed to setup store: %v", err)
				}

				var args repos.StoreListReposArgs
				if tc.args != nil {
					args = tc.args(stored)
				}

				rs, err := tx.ListRepos(ctx, args)
				if have, want := fmt.Sprint(err), fmt.Sprint(tc.err); have != want {
					t.Errorf("error:\nhave: %v\nwant: %v", have, want)
				}

				if tc.repos != nil {
					tc.repos(t, rs)
				}
			}))
		}

		t.Run("only include cloned", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			stored := mkRepos(5, repositories...).Clone()
			if err := tx.InsertRepos(ctx, stored...); err != nil {
				t.Fatalf("failed to setup store: %v", err)
			}

			sort.Sort(stored)

			cloned := stored[:3]
			if err := tx.SetClonedRepos(ctx, cloned.Names()...); err != nil {
				t.Fatalf("failed to set cloned repos: %v", err)
			}

			args := repos.StoreListReposArgs{
				ClonedOnly: true,
			}

			rs, err := tx.ListRepos(ctx, args)
			if err != nil {
				t.Errorf("failed to list repos: %v", err)
			}

			want := cloned.With(func(r *repos.Repo) {
				r.Cloned = true
			})

			repos.Assert.ReposEqual(want...)(t, rs)
		}))
	}
}

func testStoreListReposPagination(t *testing.T, store repos.Store) func(*testing.T) {
	clock := repos.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	servicesPerKind := createExternalServices(t, store)

	github := repos.Repo{
		Name:        "foo/bar",
		URI:         "github.com/foo/bar",
		Description: "The description",
		CreatedAt:   now,
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "AAAAA==",
			ServiceType: "github",
			ServiceID:   "http://github.com",
		},
		Sources: map[string]*repos.SourceInfo{
			servicesPerKind[extsvc.KindGitHub].URN(): {
				ID:       servicesPerKind[extsvc.KindGitHub].URN(),
				CloneURL: "git@github.com:foo/bar.git",
			},
		},
		Metadata: new(github.Repository),
	}

	return func(t *testing.T) {
		ctx := context.Background()
		t.Run("", transact(ctx, store, func(t testing.TB, tx repos.Store) {
			stored := mkRepos(7, &github)
			if err := tx.InsertRepos(ctx, stored...); err != nil {
				t.Fatalf("InsertRepos error: %s", err)
			}

			sort.Sort(stored)

			lo, hi := -2, len(stored)+2
			for page := lo; page < hi; page++ {
				for limit := lo; limit < hi; limit++ {
					args := repos.StoreListReposArgs{
						PerPage: int64(page),
						Limit:   int64(limit),
					}

					listed, err := tx.ListRepos(ctx, args)
					if err != nil {
						t.Fatalf("unexpected error with page=%d, limit=%d: %v", page, limit, err)
					}

					var want repos.Repos
					if limit <= 0 || limit >= len(stored) {
						want = stored
					} else {
						want = stored[:limit]
					}

					if have := repos.Repos(listed); !reflect.DeepEqual(have, want) {
						t.Fatalf("page=%d, limit=%d: %s", page, limit, cmp.Diff(have, want))
					}
				}
			}
		}))
	}
}

func testStoreListExternalRepoSpecs(db *sql.DB) func(t *testing.T, repoStore repos.Store) func(*testing.T) {
	return func(t *testing.T, store repos.Store) func(*testing.T) {
		return func(t *testing.T) {
			ctx := context.Background()

			// Insert test repositories
			_, err := db.ExecContext(ctx, `
INSERT INTO repo (id, name, description, fork, external_id, external_service_type, external_service_id, deleted_at)
VALUES
	(1, 'github.com/user/repo1', '', FALSE, NULL, 'github', 'https://github.com/', NULL),
	(2, 'github.com/user/repo2', '', FALSE, 'MDEwOlJlcG9zaXRvcnky', NULL, 'https://github.com/', NULL),
	(3, 'github.com/user/repo3', '', FALSE, 'MDEwOlJlcG9zaXRvcnkz', 'github', NULL, NULL),
	(4, 'github.com/user/repo4', '', FALSE, 'MDEwOlJlcG9zaXRvcnk0', 'github', 'https://github.com/', NOW()),
	(5, 'github.com/user/repo5', '', FALSE, 'MDEwOlJlcG9zaXRvcnk1', 'github', 'https://github.com/', NULL)
`)
			if err != nil {
				t.Fatal(err)
			}

			ids, err := store.ListExternalRepoSpecs(ctx)
			if err != nil {
				t.Fatal(err)
			}
			want := map[api.ExternalRepoSpec]struct{}{
				{
					ID:          "MDEwOlJlcG9zaXRvcnk1",
					ServiceType: "github",
					ServiceID:   "https://github.com/",
				}: {},
			}
			if diff := cmp.Diff(want, ids); diff != "" {
				t.Fatalf("IDs mismatch (-want +got):\n%s", diff)
			}
		}
	}
}

func testSyncRateLimiters(t *testing.T, store repos.Store) func(*testing.T) {
	clock := repos.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	return func(t *testing.T) {
		ctx := context.Background()
		transact(ctx, store, func(t testing.TB, tx repos.Store) {
			toCreate := 501 // Larger than default page size in order to test pagination
			services := make([]*repos.ExternalService, 0, toCreate)
			for i := 0; i < toCreate; i++ {
				svc := &repos.ExternalService{
					ID:          int64(i) + 1,
					Kind:        "GitHub",
					DisplayName: "GitHub",
					CreatedAt:   now,
					UpdatedAt:   now,
					DeletedAt:   time.Time{},
				}
				config := schema.GitLabConnection{
					Url: fmt.Sprintf("http://example%d.com/", i),
					RateLimit: &schema.GitLabRateLimit{
						RequestsPerHour: 3600,
						Enabled:         true,
					},
				}
				data, err := json.Marshal(config)
				if err != nil {
					t.Fatal(err)
				}
				svc.Config = string(data)
				services = append(services, svc)
			}

			if err := tx.UpsertExternalServices(ctx, services...); err != nil {
				t.Fatalf("failed to setup store: %v", err)
			}

			registry := ratelimit.NewRegistry()
			syncer := repos.NewRateLimitSyncer(registry, tx)
			err := syncer.SyncRateLimiters(ctx)
			if err != nil {
				t.Fatal(err)
			}
			have := registry.Count()
			if have != toCreate {
				t.Fatalf("Want %d, got %d", toCreate, have)
			}
		})(t)
	}
}

func testStoreEnqueueSyncJobs(db *sql.DB, store *repos.DBStore) func(t *testing.T, store repos.Store) func(*testing.T) {
	return func(t *testing.T, _ repos.Store) func(*testing.T) {
		t.Helper()

		clock := repos.NewFakeClock(time.Now(), 0)
		now := clock.Now()

		services := generateExternalServices(10, mkExternalServices(now)...)

		type testCase struct {
			name            string
			stored          repos.ExternalServices
			queued          func(repos.ExternalServices) []int64
			ignoreSiteAdmin bool
			err             error
		}

		var testCases []testCase

		testCases = append(testCases, testCase{
			name: "enqueue everything",
			stored: services.With(func(s *repos.ExternalService) {
				s.NextSyncAt = now.Add(-10 * time.Second)
			}),
			queued: func(svcs repos.ExternalServices) []int64 { return svcs.IDs() },
		})

		testCases = append(testCases, testCase{
			name: "nothing to enqueue",
			stored: services.With(func(s *repos.ExternalService) {
				s.NextSyncAt = now.Add(10 * time.Second)
			}),
			queued: func(svcs repos.ExternalServices) []int64 { return []int64{} },
		})

		testCases = append(testCases, testCase{
			name: "ignore siteadmin repos",
			stored: services.With(func(s *repos.ExternalService) {
				s.NextSyncAt = now.Add(10 * time.Second)
			}),
			ignoreSiteAdmin: true,
			queued:          func(svcs repos.ExternalServices) []int64 { return []int64{} },
		})

		{
			i := 0
			testCases = append(testCases, testCase{
				name: "some to enqueue",
				stored: services.With(func(s *repos.ExternalService) {
					if i%2 == 0 {
						s.NextSyncAt = now.Add(10 * time.Second)
					} else {
						s.NextSyncAt = now.Add(-10 * time.Second)
					}
					i++
				}),
				queued: func(svcs repos.ExternalServices) []int64 {
					var ids []int64
					for i := range svcs {
						if i%2 != 0 {
							ids = append(ids, svcs[i].ID)
						}
					}
					return ids
				},
			})
		}

		return func(t *testing.T) {
			ctx := context.Background()

			for _, tc := range testCases {
				tc := tc

				t.Run(tc.name, func(t *testing.T) {
					t.Cleanup(func() {
						if _, err := db.ExecContext(ctx, "DELETE FROM external_service_sync_jobs;DELETE FROM external_services"); err != nil {
							t.Fatal(err)
						}
					})
					stored := tc.stored.Clone()

					if err := store.UpsertExternalServices(ctx, stored...); err != nil {
						t.Fatalf("failed to setup store: %v", err)
					}

					err := store.EnqueueSyncJobs(ctx, tc.ignoreSiteAdmin)
					if have, want := fmt.Sprint(err), fmt.Sprint(tc.err); have != want {
						t.Errorf("error:\nhave: %v\nwant: %v", have, want)
					}

					jobs, err := store.ListSyncJobs(ctx)
					if err != nil {
						t.Fatal(err)
					}

					gotIDs := make([]int64, 0, len(jobs))
					for _, job := range jobs {
						gotIDs = append(gotIDs, job.ExternalServiceID)
					}

					want := tc.queued(stored)
					sort.Slice(gotIDs, func(i, j int) bool {
						return gotIDs[i] < gotIDs[j]
					})
					sort.Slice(want, func(i, j int) bool {
						return want[i] < want[j]
					})

					if diff := cmp.Diff(want, gotIDs); diff != "" {
						t.Fatal(diff)
					}
				})
			}
		}
	}
}

func mkRepos(n int, base ...*repos.Repo) repos.Repos {
	if len(base) == 0 {
		return nil
	}

	rs := make(repos.Repos, 0, n)
	for i := 0; i < n; i++ {
		id := strconv.Itoa(i)
		r := base[i%len(base)].Clone()
		r.Name += id
		r.ExternalRepo.ID += id
		rs = append(rs, r)
	}
	return rs
}

func generateExternalServices(n int, base ...*repos.ExternalService) repos.ExternalServices {
	if len(base) == 0 {
		return nil
	}
	es := make(repos.ExternalServices, 0, n)
	for i := 0; i < n; i++ {
		id := strconv.Itoa(i)
		r := base[i%len(base)].Clone()
		r.DisplayName += id
		es = append(es, r)
	}
	return es
}

func transact(ctx context.Context, s repos.Store, test func(testing.TB, repos.Store)) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		tr, ok := s.(repos.Transactor)

		if ok {
			txstore, err := tr.Transact(ctx)
			if err != nil {
				t.Fatalf("failed to start transaction: %v", err)
			}
			defer txstore.Done(errRollback)
			s = &noopTxStore{TB: t, Store: txstore}
		}

		test(t, s)
	}
}

type noopTxStore struct {
	testing.TB
	repos.Store
	count int
}

func (tx *noopTxStore) Transact(context.Context) (repos.TxStore, error) {
	if tx.count != 0 {
		return nil, fmt.Errorf("noopTxStore: %d current transactions", tx.count)
	}
	tx.count++
	// noop
	return tx, nil
}

func (tx *noopTxStore) Done(err error) error {
	tx.Helper()

	if tx.count != 1 {
		tx.Fatal("no current transactions")
	}
	if err != nil {
		tx.Fatal(fmt.Sprintf("unexpected error in noopTxStore: %v", err))
	}
	tx.count--

	return nil
}

func createExternalServices(t *testing.T, store repos.Store) map[string]*repos.ExternalService {
	clock := repos.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	svcs := mkExternalServices(now)

	// create a few external services
	if err := store.UpsertExternalServices(context.Background(), svcs...); err != nil {
		t.Fatalf("failed to insert external services: %v", err)
	}

	services, err := store.ListExternalServices(context.Background(), repos.StoreListExternalServicesArgs{})
	if err != nil {
		t.Fatal("failed to list external services")
	}

	servicesPerKind := make(map[string]*repos.ExternalService)
	for _, svc := range services {
		servicesPerKind[svc.Kind] = svc
	}

	return servicesPerKind
}

func mkExternalServices(now time.Time) repos.ExternalServices {
	githubSvc := repos.ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "Github - Test",
		Config:      `{"url": "https://github.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	gitlabSvc := repos.ExternalService{
		Kind:        extsvc.KindGitLab,
		DisplayName: "GitLab - Test",
		Config:      `{"url": "https://gitlab.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	bitbucketServerSvc := repos.ExternalService{
		Kind:        extsvc.KindBitbucketServer,
		DisplayName: "Bitbucket Server - Test",
		Config:      `{"url": "https://bitbucket.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	bitbucketCloudSvc := repos.ExternalService{
		Kind:        extsvc.KindBitbucketCloud,
		DisplayName: "Bitbucket Cloud - Test",
		Config:      `{"url": "https://bitbucket.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	awsSvc := repos.ExternalService{
		Kind:        extsvc.KindAWSCodeCommit,
		DisplayName: "AWS Code - Test",
		Config:      `{"url": "https://aws.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	otherSvc := repos.ExternalService{
		Kind:        extsvc.KindOther,
		DisplayName: "Other - Test",
		Config:      `{"url": "https://other.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	gitoliteSvc := repos.ExternalService{
		Kind:        extsvc.KindGitolite,
		DisplayName: "Gitolite - Test",
		Config:      `{"url": "https://gitolite.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return []*repos.ExternalService{
		&githubSvc,
		&gitlabSvc,
		&bitbucketServerSvc,
		&bitbucketCloudSvc,
		&awsSvc,
		&otherSvc,
		&gitoliteSvc,
	}
}
