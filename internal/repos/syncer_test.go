package repos_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gitchander/permutation"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/keegancsmith/sqlf"
	"github.com/lib/pq"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/awscodecommit"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketcloud"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitolite"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

func testSyncerBatchSync(s *repos.Store) func(*testing.T) {
	return testSyncerSync(s, false)
}

func testSyncerStreamingSync(s *repos.Store) func(*testing.T) {
	return testSyncerSync(s, true)
}

func testSyncerSync(s *repos.Store, streaming bool) func(*testing.T) {
	return func(t *testing.T) {
		servicesPerKind := createExternalServices(t, s)

		githubService := servicesPerKind[extsvc.KindGitHub]

		githubRepo := (&types.Repo{
			Name:     "github.com/org/foo",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-12345",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}).With(
			types.Opt.RepoSources(githubService.URN()),
		)

		gitlabService := servicesPerKind[extsvc.KindGitLab]

		gitlabRepo := (&types.Repo{
			Name:     "gitlab.com/org/foo",
			Metadata: &gitlab.Project{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "12345",
				ServiceID:   "https://gitlab.com/",
				ServiceType: extsvc.TypeGitLab,
			},
		}).With(
			types.Opt.RepoSources(gitlabService.URN()),
		)

		bitbucketServerService := servicesPerKind[extsvc.KindBitbucketServer]

		bitbucketServerRepo := (&types.Repo{
			Name:     "bitbucketserver.mycorp.com/org/foo",
			Metadata: &bitbucketserver.Repo{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "23456",
				ServiceID:   "https://bitbucketserver.mycorp.com/",
				ServiceType: "bitbucketServer",
			},
		}).With(
			types.Opt.RepoSources(bitbucketServerService.URN()),
		)

		awsCodeCommitService := servicesPerKind[extsvc.KindAWSCodeCommit]

		awsCodeCommitRepo := (&types.Repo{
			Name:     "git-codecommit.us-west-1.amazonaws.com/stripe-go",
			Metadata: &awscodecommit.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "f001337a-3450-46fd-b7d2-650c0EXAMPLE",
				ServiceID:   "arn:aws:codecommit:us-west-1:999999999999:",
				ServiceType: extsvc.TypeAWSCodeCommit,
			},
		}).With(
			types.Opt.RepoSources(awsCodeCommitService.URN()),
		)

		otherService := servicesPerKind[extsvc.KindOther]

		otherRepo := (&types.Repo{
			Name: "git-host.com/org/foo",
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "git-host.com/org/foo",
				ServiceID:   "https://git-host.com/",
				ServiceType: extsvc.TypeOther,
			},
			Metadata: &extsvc.OtherRepoMetadata{},
		}).With(
			types.Opt.RepoSources(otherService.URN()),
		)

		gitoliteService := servicesPerKind[extsvc.KindGitolite]

		gitoliteRepo := (&types.Repo{
			Name:     "gitolite.mycorp.com/foo",
			Metadata: &gitolite.Repo{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo",
				ServiceID:   "git@gitolite.mycorp.com",
				ServiceType: extsvc.TypeGitolite,
			},
		}).With(
			types.Opt.RepoSources(gitoliteService.URN()),
		)

		bitbucketCloudService := servicesPerKind[extsvc.KindBitbucketCloud]

		bitbucketCloudRepo := (&types.Repo{
			Name:     "bitbucket.org/team/foo",
			Metadata: &bitbucketcloud.Repo{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "{e164a64c-bd73-4a40-b447-d71b43f328a8}",
				ServiceID:   "https://bitbucket.org/",
				ServiceType: extsvc.TypeBitbucketCloud,
			},
		}).With(
			types.Opt.RepoSources(bitbucketCloudService.URN()),
		)

		clock := timeutil.NewFakeClock(time.Now(), 0)

		svcdup := types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github2 - Test",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   clock.Now(),
			UpdatedAt:   clock.Now(),
		}

		// create a few external services
		if err := s.ExternalServiceStore.Upsert(context.Background(), &svcdup); err != nil {
			t.Fatalf("failed to insert external services: %v", err)
		}

		type testCase struct {
			name    string
			sourcer repos.Sourcer
			store   *repos.Store
			stored  types.Repos
			svcs    []*types.ExternalService
			ctx     context.Context
			now     func() time.Time
			diff    repos.Diff
			err     string
		}

		var testCases []testCase
		for _, tc := range []struct {
			repo *types.Repo
			svc  *types.ExternalService
		}{
			{repo: githubRepo, svc: githubService},
			{repo: gitlabRepo, svc: gitlabService},
			{repo: bitbucketServerRepo, svc: bitbucketServerService},
			{repo: awsCodeCommitRepo, svc: awsCodeCommitService},
			{repo: otherRepo, svc: otherService},
			{repo: gitoliteRepo, svc: gitoliteService},
			{repo: bitbucketCloudRepo, svc: bitbucketCloudService},
		} {
			testCases = append(testCases,
				testCase{
					name: string(tc.repo.Name) + "/new repo",
					sourcer: repos.NewFakeSourcer(nil,
						repos.NewFakeSource(tc.svc.Clone(), nil, tc.repo.Clone()),
					),
					store:  s,
					stored: types.Repos{},
					now:    clock.Now,
					diff: repos.Diff{Added: types.Repos{tc.repo.With(
						types.Opt.RepoCreatedAt(clock.Time(1)),
						types.Opt.RepoSources(tc.svc.Clone().URN()),
					)}},
					svcs: []*types.ExternalService{tc.svc},
					err:  "<nil>",
				},
				testCase{
					// If the source is unauthorized we should treat this as if zero repos were
					// returned as it indicates that the source no longer has access to its repos
					name: string(tc.repo.Name) + "/unauthorized",
					sourcer: repos.NewFakeSourcer(nil,
						repos.NewFakeSource(tc.svc.Clone(), &repos.ErrUnauthorized{}),
					),
					store: s,
					stored: types.Repos{tc.repo.With(
						types.Opt.RepoSources(tc.svc.URN()),
					)},
					now: clock.Now,
					diff: repos.Diff{Deleted: types.Repos{tc.repo.With(
						types.Opt.RepoSources(tc.svc.URN(), svcdup.URN()),
					)}},
					svcs: []*types.ExternalService{tc.svc},
					err:  "bad credentials",
				},
				testCase{
					// If the source is forbidden we should treat this as if zero repos were returned
					// as it indicates that the source no longer has access to its repos
					name: string(tc.repo.Name) + "/forbidden",
					sourcer: repos.NewFakeSourcer(nil,
						repos.NewFakeSource(tc.svc.Clone(), &repos.ErrForbidden{}),
					),
					store: s,
					stored: types.Repos{tc.repo.With(
						types.Opt.RepoSources(tc.svc.URN()),
					)},
					now: clock.Now,
					diff: repos.Diff{Deleted: types.Repos{tc.repo.With(
						types.Opt.RepoSources(tc.svc.URN(), svcdup.URN()),
					)}},
					svcs: []*types.ExternalService{tc.svc},
					err:  "forbidden",
				},
				testCase{
					// If the source account has been suspended we should treat this as if zero repos were returned as it indicates
					// that the source no longer has access to its repos
					name: string(tc.repo.Name) + "/accountsuspended",
					sourcer: repos.NewFakeSourcer(nil,
						repos.NewFakeSource(tc.svc.Clone(), &repos.ErrAccountSuspended{}),
					),
					store: s,
					stored: types.Repos{tc.repo.With(
						types.Opt.RepoSources(tc.svc.URN()),
					)},
					now: clock.Now,
					diff: repos.Diff{Deleted: types.Repos{tc.repo.With(
						types.Opt.RepoSources(tc.svc.URN(), svcdup.URN()),
					)}},
					svcs: []*types.ExternalService{tc.svc},
					err:  "account suspended",
				},
				testCase{
					// It's expected that there could be multiple stored sources but only one will ever be returned
					// by the code host as it can't know about others.
					name: string(tc.repo.Name) + "/source already stored",
					sourcer: repos.NewFakeSourcer(nil,
						repos.NewFakeSource(tc.svc.Clone(), nil, tc.repo.Clone()),
					),
					store: s,
					stored: types.Repos{tc.repo.With(
						types.Opt.RepoSources(tc.svc.URN(), svcdup.URN()),
					)},
					now: clock.Now,
					diff: repos.Diff{Unmodified: types.Repos{tc.repo.With(
						types.Opt.RepoSources(tc.svc.URN(), svcdup.URN()),
					)}},
					svcs: []*types.ExternalService{tc.svc},
					err:  "<nil>",
				},
				testCase{
					name:    string(tc.repo.Name) + "/deleted ALL repo sources",
					sourcer: repos.NewFakeSourcer(nil),
					store:   s,
					stored: types.Repos{tc.repo.With(
						types.Opt.RepoSources(tc.svc.URN(), svcdup.URN()),
					)},
					now: clock.Now,
					diff: repos.Diff{Deleted: types.Repos{tc.repo.With(
						types.Opt.RepoDeletedAt(clock.Time(1)),
					)}},
					svcs: []*types.ExternalService{tc.svc, &svcdup},
					err:  "<nil>",
				},
				testCase{
					name:    string(tc.repo.Name) + "/renamed repo is detected via external_id",
					sourcer: repos.NewFakeSourcer(nil, repos.NewFakeSource(tc.svc.Clone(), nil, tc.repo.Clone())),
					store:   s,
					stored: types.Repos{tc.repo.With(func(r *types.Repo) {
						r.Name = "old-name"
					})},
					now: clock.Now,
					diff: repos.Diff{Modified: types.Repos{
						tc.repo.With(
							types.Opt.RepoModifiedAt(clock.Time(1))),
					}},
					svcs: []*types.ExternalService{tc.svc},
					err:  "<nil>",
				},
				testCase{
					name: string(tc.repo.Name) + "/repo got renamed to another repo that gets deleted",
					sourcer: repos.NewFakeSourcer(nil,
						repos.NewFakeSource(tc.svc.Clone(), nil,
							tc.repo.With(func(r *types.Repo) { r.ExternalRepo.ID = "another-id" }),
						),
					),
					store: s,
					stored: types.Repos{
						tc.repo.Clone(),
						tc.repo.With(func(r *types.Repo) {
							r.Name = "another-repo"
							r.ExternalRepo.ID = "another-id"
						}),
					},
					now: clock.Now,
					diff: repos.Diff{
						Deleted: types.Repos{
							tc.repo.With(func(r *types.Repo) {
								r.Sources = map[string]*types.SourceInfo{}
								r.DeletedAt = clock.Time(0)
								r.UpdatedAt = clock.Time(0)
							}),
						},
						Modified: types.Repos{
							tc.repo.With(
								types.Opt.RepoModifiedAt(clock.Time(1)),
								func(r *types.Repo) { r.ExternalRepo.ID = "another-id" },
							),
						},
					},
					svcs: []*types.ExternalService{tc.svc},
					err:  "<nil>",
				},
				testCase{
					name: string(tc.repo.Name) + "/repo inserted with same name as another repo that gets deleted",
					sourcer: repos.NewFakeSourcer(nil,
						repos.NewFakeSource(tc.svc.Clone(), nil,
							tc.repo,
						),
					),
					store: s,
					stored: types.Repos{
						tc.repo.With(types.Opt.RepoExternalID("another-id")),
					},
					now: clock.Now,
					diff: repos.Diff{
						Added: types.Repos{
							tc.repo.With(
								types.Opt.RepoCreatedAt(clock.Time(1)),
								types.Opt.RepoModifiedAt(clock.Time(1)),
							),
						},
						Deleted: types.Repos{
							tc.repo.With(func(r *types.Repo) {
								r.ExternalRepo.ID = "another-id"
								r.Sources = map[string]*types.SourceInfo{}
								r.DeletedAt = clock.Time(0)
								r.UpdatedAt = clock.Time(0)
							}),
						},
					},
					svcs: []*types.ExternalService{tc.svc},
					err:  "<nil>",
				},
				testCase{
					name: string(tc.repo.Name) + "/repo inserted with same name as repo without id",
					sourcer: repos.NewFakeSourcer(nil,
						repos.NewFakeSource(tc.svc.Clone(), nil,
							tc.repo,
						),
					),
					store: s,
					stored: types.Repos{
						tc.repo.With(types.Opt.RepoName("old-name")),  // same external id as sourced
						tc.repo.With(types.Opt.RepoExternalID("bar")), // same name as sourced
					}.With(types.Opt.RepoCreatedAt(clock.Time(1))),
					now: clock.Now,
					diff: repos.Diff{
						Modified: types.Repos{
							tc.repo.With(
								types.Opt.RepoCreatedAt(clock.Time(1)),
								types.Opt.RepoModifiedAt(clock.Time(1)),
							),
						},
						Deleted: types.Repos{
							tc.repo.With(func(r *types.Repo) {
								r.ExternalRepo.ID = ""
								r.Sources = map[string]*types.SourceInfo{}
								r.DeletedAt = clock.Time(0)
								r.UpdatedAt = clock.Time(0)
								r.CreatedAt = clock.Time(0)
							}),
						},
					},
					svcs: []*types.ExternalService{tc.svc},
					err:  "<nil>",
				},
				testCase{
					name:    string(tc.repo.Name) + "/renamed repo which was deleted is detected and added",
					sourcer: repos.NewFakeSourcer(nil, repos.NewFakeSource(tc.svc.Clone(), nil, tc.repo.Clone())),
					store:   s,
					stored: types.Repos{tc.repo.With(func(r *types.Repo) {
						r.Sources = map[string]*types.SourceInfo{}
						r.Name = "old-name"
						r.DeletedAt = clock.Time(0)
					})},
					now: clock.Now,
					diff: repos.Diff{Added: types.Repos{
						tc.repo.With(
							types.Opt.RepoCreatedAt(clock.Time(1))),
					}},
					svcs: []*types.ExternalService{tc.svc},
					err:  "<nil>",
				},
				testCase{
					name: string(tc.repo.Name) + "/repos have their names swapped",
					sourcer: repos.NewFakeSourcer(nil, repos.NewFakeSource(tc.svc.Clone(), nil,
						tc.repo.With(func(r *types.Repo) {
							r.Name = "foo"
							r.ExternalRepo.ID = "1"
						}),
						tc.repo.With(func(r *types.Repo) {
							r.Name = "bar"
							r.ExternalRepo.ID = "2"
						}),
					)),
					now:   clock.Now,
					store: s,
					stored: types.Repos{
						tc.repo.With(func(r *types.Repo) {
							r.Name = "bar"
							r.ExternalRepo.ID = "1"
						}),
						tc.repo.With(func(r *types.Repo) {
							r.Name = "foo"
							r.ExternalRepo.ID = "2"
						}),
					},
					diff: repos.Diff{
						Modified: types.Repos{
							tc.repo.With(func(r *types.Repo) {
								r.Name = "foo"
								r.ExternalRepo.ID = "1"
								r.UpdatedAt = clock.Time(0)
							}),
							tc.repo.With(func(r *types.Repo) {
								r.Name = "bar"
								r.ExternalRepo.ID = "2"
								r.UpdatedAt = clock.Time(0)
							}),
						},
					},
					svcs: []*types.ExternalService{tc.svc},
					err:  "<nil>",
				},
				testCase{
					name: string(tc.repo.Name) + "/case insensitive name",
					sourcer: repos.NewFakeSourcer(nil, repos.NewFakeSource(tc.svc.Clone(), nil,
						tc.repo.Clone(),
						tc.repo.With(types.Opt.RepoName(api.RepoName(strings.ToUpper(string(tc.repo.Name))))),
					)),
					store:  s,
					stored: types.Repos{tc.repo.With(types.Opt.RepoName(api.RepoName(strings.ToUpper(string(tc.repo.Name)))))},
					now:    clock.Now,
					diff:   repos.Diff{Modified: types.Repos{tc.repo.With(types.Opt.RepoModifiedAt(clock.Time(0)))}},
					svcs:   []*types.ExternalService{tc.svc},
					err:    "<nil>",
				},
			)

			if !streaming {
				// streaming SyncExternalService removes the possibility of adding any sources to a repo beyond the
				// currently syncing external service, so we don't test this for the streaming syncer.
				testCases = append(testCases, testCase{
					name: string(tc.repo.Name) + "/new repo sources",
					sourcer: repos.NewFakeSourcer(nil,
						repos.NewFakeSource(tc.svc.Clone(), nil, tc.repo.Clone()),
						repos.NewFakeSource(svcdup.Clone(), nil, tc.repo.Clone()),
					),
					store:  s,
					stored: types.Repos{tc.repo.Clone()},
					now:    clock.Now,
					diff: repos.Diff{Modified: types.Repos{tc.repo.With(
						types.Opt.RepoModifiedAt(clock.Time(1)),
						types.Opt.RepoSources(tc.svc.URN(), svcdup.URN()),
					)}},
					svcs: []*types.ExternalService{tc.svc},
					err:  "<nil>",
				})
			}
		}

		for _, tc := range testCases {
			if tc.name == "" {
				continue
			}

			tc := tc
			ctx := context.Background()

			t.Run(tc.name, transact(ctx, tc.store, func(t testing.TB, st *repos.Store) {
				defer func() {
					if err := recover(); err != nil {
						t.Fatalf("%q panicked: %v", tc.name, err)
					}
				}()

				now := tc.now
				if now == nil {
					clock := timeutil.NewFakeClock(time.Now(), time.Second)
					now = clock.Now
				}

				ctx := tc.ctx
				if ctx == nil {
					ctx = context.Background()
				}

				if st != nil && len(tc.stored) > 0 {
					cloned := tc.stored.Clone()
					if err := st.RepoStore.Create(ctx, cloned...); err != nil {
						t.Fatalf("failed to prepare store: %v", err)
					}
				}

				syncer := &repos.Syncer{
					Sourcer:   tc.sourcer,
					Store:     tc.store,
					Now:       now,
					Streaming: streaming,
				}

				for _, svc := range tc.svcs {
					err := syncer.SyncExternalService(ctx, st, svc.ID, time.Millisecond)

					if have, want := fmt.Sprint(err), tc.err; !strings.Contains(have, want) {
						t.Errorf("error %q doesn't contain %q", have, want)
					}
				}

				if st != nil {
					var want, have types.Repos
					want.Concat(tc.diff.Added, tc.diff.Modified, tc.diff.Unmodified)
					have, _ = st.RepoStore.List(ctx, database.ReposListOptions{})

					want = want.With(types.Opt.RepoID(0))
					have = have.With(types.Opt.RepoID(0))
					sort.Sort(want)
					sort.Sort(have)

					types.Assert.ReposEqual(want...)(t, have)
				}
			}))
		}
	}
}

func testBatchSyncRepo(s *repos.Store) func(*testing.T) {
	return testSyncRepo(s, false)
}

func testStreamingSyncRepo(s *repos.Store) func(*testing.T) {
	return testSyncRepo(s, true)
}

func testSyncRepo(s *repos.Store, streaming bool) func(*testing.T) {
	return func(t *testing.T) {
		clock := timeutil.NewFakeClock(time.Now(), time.Second)

		opts := []func(*types.ExternalService){}
		if streaming {
			// Streaming SyncRepo only syncs repos if it finds a cloud default external service of the same type.
			opts = append(opts, func(svc *types.ExternalService) { svc.CloudDefault = true })
		}

		servicesPerKind := createExternalServices(t, s, opts...)

		repo := &types.Repo{
			ID:          0, // explicitly make default value for sourced repo
			Name:        "github.com/foo/bar",
			Description: "The description",
			Archived:    false,
			Fork:        false,
			Stars:       100,
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
				ServiceType: extsvc.TypeGitHub,
				ServiceID:   "https://github.com/",
			},
			Sources: map[string]*types.SourceInfo{
				servicesPerKind[extsvc.KindGitHub].URN(): {
					ID:       servicesPerKind[extsvc.KindGitHub].URN(),
					CloneURL: "git@github.com:foo/bar.git",
				},
			},
			Metadata: &github.Repository{
				ID:             "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
				URL:            "github.com/foo/bar",
				DatabaseID:     1234,
				Description:    "The description",
				NameWithOwner:  "foo/bar",
				StargazerCount: 100,
			},
		}

		testCases := []struct {
			name    string
			sourced *types.Repo
			stored  types.Repos
			assert  types.ReposAssertion
		}{{
			name:    "insert",
			sourced: repo,
			assert:  types.Assert.ReposEqual(repo.With(types.Opt.RepoCreatedAt(clock.Time(2)))),
		}, {
			name:    "update",
			sourced: repo,
			stored: types.Repos{
				repo.With(
					types.Opt.RepoCreatedAt(clock.Time(2)),
					func(r *types.Repo) { r.Stars = 0 },
				),
			},
			assert: types.Assert.ReposEqual(repo.With(
				types.Opt.RepoModifiedAt(clock.Time(2)),
				types.Opt.RepoCreatedAt(clock.Time(2)))),
		}, {
			name:    "update name",
			sourced: repo,
			stored: types.Repos{repo.With(
				types.Opt.RepoName("old/name"),
				types.Opt.RepoCreatedAt(clock.Time(2)))},
			assert: types.Assert.ReposEqual(repo.With(
				types.Opt.RepoModifiedAt(clock.Time(2)),
				types.Opt.RepoCreatedAt(clock.Time(2)))),
		}, {
			name:    "delete conflicting name",
			sourced: repo,
			stored: types.Repos{repo.With(
				types.Opt.RepoExternalID("old id"),
				types.Opt.RepoCreatedAt(clock.Time(2)))},
			assert: types.Assert.ReposEqual(repo.With(
				types.Opt.RepoCreatedAt(clock.Time(2)))),
		}, {
			name:    "rename and delete conflicting name",
			sourced: repo,
			stored: types.Repos{
				repo.With(
					types.Opt.RepoExternalID("old id"),
					types.Opt.RepoCreatedAt(clock.Time(2))),
				repo.With(
					types.Opt.RepoName("old name"),
					types.Opt.RepoCreatedAt(clock.Time(2))),
			},
			assert: types.Assert.ReposEqual(repo.With(
				types.Opt.RepoCreatedAt(clock.Time(2)))),
		}}

		for _, tc := range testCases {
			if tc.name == "" {
				continue
			}

			tc := tc
			ctx := context.Background()

			t.Run(tc.name, transact(ctx, s, func(t testing.TB, st *repos.Store) {
				if len(tc.stored) > 0 {
					if err := st.RepoStore.Create(ctx, tc.stored.Clone()...); err != nil {
						t.Fatalf("failed to prepare store: %v", err)
					}
				}

				clock := clock
				syncer := &repos.Syncer{
					Now:       clock.Now,
					Store:     st,
					Streaming: streaming,
				}
				err := syncer.SyncRepo(ctx, st, tc.sourced.Clone())
				if err != nil {
					t.Fatal(err)
				}

				have, err := st.RepoStore.List(ctx, database.ReposListOptions{})
				if err != nil {
					t.Fatal(err)
				}

				tc.assert(t, have)
			}))
		}
	}
}

func TestDiff(t *testing.T) {
	t.Parallel()

	eid := func(id string) api.ExternalRepoSpec {
		return api.ExternalRepoSpec{
			ID:          id,
			ServiceType: "fake",
			ServiceID:   "https://fake.com",
		}
	}
	now := time.Now()

	type testCase struct {
		name   string
		store  types.Repos
		source types.Repos
		diff   repos.Diff
	}

	cases := []testCase{
		{
			name: "empty",
			diff: repos.Diff{},
		},
		{
			name:   "added",
			source: types.Repos{{ExternalRepo: eid("1")}},
			diff:   repos.Diff{Added: types.Repos{{ExternalRepo: eid("1")}}},
		},
		{
			name:  "deleted",
			store: types.Repos{{ExternalRepo: eid("1")}},
			diff:  repos.Diff{Deleted: types.Repos{{ExternalRepo: eid("1")}}},
		},
		{
			name: "modified",
			store: types.Repos{
				{ExternalRepo: eid("1"), Name: "foo", Description: "foo"},
				{ExternalRepo: eid("2"), Name: "bar"},
			},
			source: types.Repos{
				{ExternalRepo: eid("1"), Name: "foo", Description: "bar"},
				{ExternalRepo: eid("2"), Name: "bar", URI: "2"},
			},
			diff: repos.Diff{Modified: types.Repos{
				{ExternalRepo: eid("1"), Name: "foo", Description: "bar"},
				{ExternalRepo: eid("2"), Name: "bar", URI: "2"},
			}},
		},
		{
			name:   "unmodified",
			store:  types.Repos{{ExternalRepo: eid("1"), Description: "foo"}},
			source: types.Repos{{ExternalRepo: eid("1"), Description: "foo"}},
			diff: repos.Diff{Unmodified: types.Repos{
				{ExternalRepo: eid("1"), Description: "foo"},
			}},
		},
		{
			name: "duplicates in source are merged",
			source: types.Repos{
				{ExternalRepo: eid("1"), Description: "foo", Sources: map[string]*types.SourceInfo{
					"a": {ID: "a"},
				}},
				{ExternalRepo: eid("1"), Description: "bar", Sources: map[string]*types.SourceInfo{
					"b": {ID: "b"},
				}},
			},
			diff: repos.Diff{Added: types.Repos{
				{ExternalRepo: eid("1"), Description: "bar", Sources: map[string]*types.SourceInfo{
					"a": {ID: "a"},
					"b": {ID: "b"},
				}},
			}},
		},
		{
			name: "duplicate with a changed name is merged correctly",
			store: types.Repos{
				{Name: "1", ExternalRepo: eid("1"), Description: "foo"},
			},
			source: types.Repos{
				{Name: "2", ExternalRepo: eid("1"), Description: "foo"},
			},
			diff: repos.Diff{Modified: types.Repos{
				{Name: "2", ExternalRepo: eid("1"), Description: "foo"},
			}},
		},
		{
			name: "unmodified preserves stored repo",
			store: types.Repos{
				{ExternalRepo: eid("1"), Description: "foo", UpdatedAt: now},
			},
			source: types.Repos{
				{ExternalRepo: eid("1"), Description: "foo"},
			},
			diff: repos.Diff{Unmodified: types.Repos{
				{ExternalRepo: eid("1"), Description: "foo", UpdatedAt: now},
			}},
		},
	}

	permutedCases := []testCase{
		{
			// Repo renamed and repo created with old name
			name: "renamed repo",
			store: types.Repos{
				{Name: "1", ExternalRepo: eid("old"), Description: "foo"},
			},
			source: types.Repos{
				{Name: "2", ExternalRepo: eid("old"), Description: "foo"},
				{Name: "1", ExternalRepo: eid("new"), Description: "bar"},
			},
			diff: repos.Diff{
				Modified: types.Repos{
					{Name: "2", ExternalRepo: eid("old"), Description: "foo"},
				},
				Added: types.Repos{
					{Name: "1", ExternalRepo: eid("new"), Description: "bar"},
				},
			},
		},
		{
			name:  "repo renamed to an already deleted repo",
			store: types.Repos{},
			source: types.Repos{
				{Name: "a", ExternalRepo: eid("b")},
			},
			diff: repos.Diff{
				Added: types.Repos{
					{Name: "a", ExternalRepo: eid("b")},
				},
			},
		},
		{
			name: "repo renamed to a repo that gets deleted",
			store: types.Repos{
				{Name: "a", ExternalRepo: eid("a")},
				{Name: "b", ExternalRepo: eid("b")},
			},
			source: types.Repos{
				{Name: "a", ExternalRepo: eid("b")},
			},
			diff: repos.Diff{
				Deleted:  types.Repos{{Name: "a", ExternalRepo: eid("a")}},
				Modified: types.Repos{{Name: "a", ExternalRepo: eid("b")}},
			},
		},
		{
			name: "swapped repo",
			store: types.Repos{
				{Name: "foo", ExternalRepo: eid("1"), Description: "foo"},
				{Name: "bar", ExternalRepo: eid("2"), Description: "bar"},
			},
			source: types.Repos{
				{Name: "bar", ExternalRepo: eid("1"), Description: "bar"},
				{Name: "foo", ExternalRepo: eid("2"), Description: "foo"},
			},
			diff: repos.Diff{
				Modified: types.Repos{
					{Name: "bar", ExternalRepo: eid("1"), Description: "bar"},
					{Name: "foo", ExternalRepo: eid("2"), Description: "foo"},
				},
			},
		},
		{
			name: "deterministic merging of source",
			source: types.Repos{
				{Name: "foo", ExternalRepo: eid("1"), Description: "desc1", Sources: map[string]*types.SourceInfo{"a": nil}},
				{Name: "foo", ExternalRepo: eid("1"), Description: "desc2", Sources: map[string]*types.SourceInfo{"b": nil}},
			},
			diff: repos.Diff{
				Added: types.Repos{
					{Name: "foo", ExternalRepo: eid("1"), Description: "desc2", Sources: map[string]*types.SourceInfo{"a": nil, "b": nil}},
				},
			},
		},
		{
			name: "conflict on case insensitive name",
			source: types.Repos{
				{Name: "foo", ExternalRepo: eid("1")},
				{Name: "Foo", ExternalRepo: eid("2")},
			},
			diff: repos.Diff{
				Added: types.Repos{
					{Name: "Foo", ExternalRepo: eid("2")},
				},
			},
		},
		{
			name: "conflict on case insensitive name exists 1",
			store: types.Repos{
				{Name: "foo", ExternalRepo: eid("1")},
			},
			source: types.Repos{
				{Name: "foo", ExternalRepo: eid("1")},
				{Name: "Foo", ExternalRepo: eid("2")},
			},
			diff: repos.Diff{
				Added: types.Repos{
					{Name: "Foo", ExternalRepo: eid("2")},
				},
				Deleted: types.Repos{
					{Name: "foo", ExternalRepo: eid("1")},
				},
			},
		},
		{
			name: "conflict on case insensitive name exists 2",
			store: types.Repos{
				{Name: "Foo", ExternalRepo: eid("2")},
			},
			source: types.Repos{
				{Name: "foo", ExternalRepo: eid("1")},
				{Name: "Foo", ExternalRepo: eid("2")},
			},
			diff: repos.Diff{
				Unmodified: types.Repos{
					{Name: "Foo", ExternalRepo: eid("2")},
				},
			},
		},
		// 🚨 SECURITY: Tests to ensure we detect repository visibility changes.
		{
			name: "repository visiblity changed",
			store: types.Repos{
				{Name: "foo", ExternalRepo: eid("1"), Description: "foo", Private: false},
			},
			source: types.Repos{
				{Name: "foo", ExternalRepo: eid("1"), Description: "foo", Private: true},
			},
			diff: repos.Diff{
				Modified: types.Repos{
					{Name: "foo", ExternalRepo: eid("1"), Description: "foo", Private: true},
				},
			},
		},
	}

	for _, tc := range permutedCases {
		store := permutation.New(tc.store)
		for i := 0; store.Next(); i++ {
			source := permutation.New(tc.source)
			for j := 0; source.Next(); j++ {
				cases = append(cases, testCase{
					name:   fmt.Sprintf("%s/permutation_%d_%d", tc.name, i, j),
					store:  tc.store.Clone(),
					source: tc.source.Clone(),
					diff:   tc.diff,
				})
			}
		}
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			diff := repos.NewDiff(tc.source, tc.store)
			diff.Sort()
			tc.diff.Sort()
			if cDiff := cmp.Diff(diff, tc.diff); cDiff != "" {
				t.Fatalf("unexpected diff:\n%s", cDiff)
			}
		})
	}
}

func testBatchSyncRun(store *repos.Store) func(t *testing.T) {
	return testSyncRun(store, false)
}

func testStreamingSyncRun(store *repos.Store) func(t *testing.T) {
	return testSyncRun(store, true)
}

func testSyncRun(store *repos.Store, streaming bool) func(t *testing.T) {
	return func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		svc := &types.ExternalService{
			Config: `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "abc"}`,
			Kind:   extsvc.KindGitHub,
		}

		if err := store.ExternalServiceStore.Upsert(ctx, svc); err != nil {
			t.Fatal(err)
		}

		mk := func(name string) *types.Repo {
			return &types.Repo{
				Name:     api.RepoName(name),
				Metadata: &github.Repository{},
				ExternalRepo: api.ExternalRepoSpec{
					ID:          name,
					ServiceID:   "https://github.com",
					ServiceType: svc.Kind,
				},
			}
		}

		// Our test will have 1 initial repo, and discover a new repo on sourcing.
		stored := types.Repos{mk("initial")}.With(types.Opt.RepoSources(svc.URN()))
		sourced := types.Repos{
			mk("initial").With(func(r *types.Repo) { r.Description = "updated" }),
			mk("new"),
		}

		syncer := &repos.Syncer{
			Sourcer:          repos.NewFakeSourcer(nil, repos.NewFakeSource(svc, nil, sourced...)),
			Store:            store,
			Synced:           make(chan repos.Diff),
			SingleRepoSynced: make(chan repos.Diff),
			Now:              time.Now,
			Streaming:        streaming,
		}

		// Initial repos in store
		if err := store.RepoStore.Create(ctx, stored...); err != nil {
			t.Fatal(err)
		}

		done := make(chan error)
		go func() {
			done <- syncer.Run(ctx, store, repos.RunOptions{
				EnqueueInterval: func() time.Duration { return time.Second },
				IsCloud:         false,
				MinSyncInterval: func() time.Duration { return 1 * time.Millisecond },
				DequeueInterval: 1 * time.Millisecond,
			})
		}()

		// Ignore fields store adds
		ignore := cmpopts.IgnoreFields(types.Repo{}, "ID", "CreatedAt", "UpdatedAt", "Sources")

		// The first thing sent down Synced is the list of repos in store.
		diff := <-syncer.Synced
		if d := cmp.Diff(repos.Diff{Unmodified: stored}, diff, ignore); d != "" {
			t.Fatalf("Synced mismatch (-want +got):\n%s", d)
		}

		if !streaming {
			// Next up it should find the new repo and send it down SingleRepoSynced
			diff = <-syncer.SingleRepoSynced
			if d := cmp.Diff(repos.Diff{Added: types.Repos{mk("new")}}, diff, ignore); d != "" {
				t.Fatalf("Synced mismatch (-want +got):\n%s", d)
			}

			// Finally we get the final diff.
			diff = <-syncer.Synced
			if d := cmp.Diff(repos.Diff{Modified: sourced[:1], Unmodified: sourced[1:]}, diff, ignore); d != "" {
				t.Fatalf("Synced mismatch (-want +got):\n%s", d)
			}

			// We check synced again to test us going around the Run loop 2 times in
			// total.
			diff = <-syncer.Synced
			if d := cmp.Diff(repos.Diff{Unmodified: sourced}, diff, ignore); d != "" {
				t.Fatalf("Synced mismatch (-want +got):\n%s", d)
			}
		} else {
			// Next up it should find the existing repo and send it down Synced
			diff = <-syncer.Synced
			if d := cmp.Diff(repos.Diff{Modified: sourced[:1]}, diff, ignore); d != "" {
				t.Fatalf("Synced mismatch (-want +got):\n%s", d)
			}

			// Then the new repo.
			diff = <-syncer.Synced
			if d := cmp.Diff(repos.Diff{Added: sourced[1:]}, diff, ignore); d != "" {
				t.Fatalf("Synced mismatch (-want +got):\n%s", d)
			}

			// We check synced again to test us going around the Run loop 2 times in
			// total.
			diff = <-syncer.Synced
			if d := cmp.Diff(repos.Diff{Unmodified: sourced[:1]}, diff, ignore); d != "" {
				t.Fatalf("Synced mismatch (-want +got):\n%s", d)
			}

			diff = <-syncer.Synced
			if d := cmp.Diff(repos.Diff{Unmodified: sourced[1:]}, diff, ignore); d != "" {
				t.Fatalf("Synced mismatch (-want +got):\n%s", d)
			}
		}

		// Cancel context and the run loop should stop
		cancel()
		err := <-done
		if err != nil && err != context.Canceled {
			t.Fatal(err)
		}
	}
}

func testBatchSyncerMultipleServices(store *repos.Store) func(t *testing.T) {
	return testSyncerMultipleServices(store, false)
}

func testStreamingSyncerMultipleServices(store *repos.Store) func(t *testing.T) {
	return testSyncerMultipleServices(store, true)
}

func testSyncerMultipleServices(store *repos.Store, streaming bool) func(t *testing.T) {
	return func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		services := mkExternalServices(time.Now())

		githubService := services[0]
		gitlabService := services[1]
		bitbucketCloudService := services[3]

		services = types.ExternalServices{
			githubService,
			gitlabService,
			bitbucketCloudService,
		}

		// setup services
		if err := store.ExternalServiceStore.Upsert(ctx, services...); err != nil {
			t.Fatal(err)
		}

		githubRepo := (&types.Repo{
			Name:     "github.com/org/foo",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-12345",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}).With(
			types.Opt.RepoSources(githubService.URN()),
		)

		gitlabRepo := (&types.Repo{
			Name:     "gitlab.com/org/foo",
			Metadata: &gitlab.Project{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "12345",
				ServiceID:   "https://gitlab.com/",
				ServiceType: extsvc.TypeGitLab,
			},
		}).With(
			types.Opt.RepoSources(gitlabService.URN()),
		)

		bitbucketCloudRepo := (&types.Repo{
			Name:     "bitbucket.org/team/foo",
			Metadata: &bitbucketcloud.Repo{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "{e164a64c-bd73-4a40-b447-d71b43f328a8}",
				ServiceID:   "https://bitbucket.org/",
				ServiceType: extsvc.TypeBitbucketCloud,
			},
		}).With(
			types.Opt.RepoSources(bitbucketCloudService.URN()),
		)

		removeSources := func(r *types.Repo) {
			r.Sources = nil
		}

		baseGithubRepos := mkRepos(10, githubRepo)
		githubSourced := baseGithubRepos.Clone().With(removeSources)
		baseGitlabRepos := mkRepos(10, gitlabRepo)
		gitlabSourced := baseGitlabRepos.Clone().With(removeSources)
		baseBitbucketCloudRepos := mkRepos(10, bitbucketCloudRepo)
		bitbucketCloudSourced := baseBitbucketCloudRepos.Clone().With(removeSources)

		sourcers := map[int64]repos.Source{
			githubService.ID:         repos.NewFakeSource(githubService, nil, githubSourced...),
			gitlabService.ID:         repos.NewFakeSource(gitlabService, nil, gitlabSourced...),
			bitbucketCloudService.ID: repos.NewFakeSource(bitbucketCloudService, nil, bitbucketCloudSourced...),
		}

		syncer := &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				if len(services) > 1 {
					t.Fatalf("Expected 1 service, got %d", len(services))
				}
				s, ok := sourcers[services[0].ID]
				if !ok {
					t.Fatalf("sourcer not found: %d", services[0].ID)
				}
				return repos.Sources{s}, nil
			},
			Store:     store,
			Synced:    make(chan repos.Diff),
			Now:       time.Now,
			Streaming: streaming,
		}

		done := make(chan error)
		go func() {
			done <- syncer.Run(ctx, store, repos.RunOptions{
				EnqueueInterval: func() time.Duration { return time.Second },
				IsCloud:         false,
				MinSyncInterval: func() time.Duration { return 1 * time.Minute },
				DequeueInterval: 1 * time.Millisecond,
			})
		}()

		// Ignore fields store adds
		ignore := cmpopts.IgnoreFields(types.Repo{}, "ID", "CreatedAt", "UpdatedAt", "Sources")

		// The first thing sent down Synced is an empty list of repos in store.
		diff := <-syncer.Synced
		if d := cmp.Diff(repos.Diff{}, diff, ignore); d != "" {
			t.Fatalf("initial Synced mismatch (-want +got):\n%s", d)
		}

		// it should add a job for all external services
		var jobCount int
		for i := 0; i < 10; i++ {
			if err := store.QueryRow(ctx, sqlf.Sprintf("SELECT COUNT(*) FROM external_service_sync_jobs")).Scan(&jobCount); err != nil {
				t.Fatal(err)
			}
			if jobCount == len(services) {
				break
			}
			// We need to give the worker package time to create the jobs
			time.Sleep(10 * time.Millisecond)
		}
		if jobCount != len(services) {
			t.Fatalf("expected %d sync jobs, got %d", len(services), jobCount)
		}

		if !streaming {
			for i := 0; i < len(services); i++ {
				diff = <-syncer.Synced
				if len(diff.Added) != 10 {
					t.Fatalf("Expected 10 Added repos. got %d", len(diff.Added))
				}
				if len(diff.Deleted) != 0 {
					t.Fatalf("Expected 0 Deleted repos. got %d", len(diff.Added))
				}
				if len(diff.Modified) != 0 {
					t.Fatalf("Expected 0 Modified repos. got %d", len(diff.Added))
				}
				if len(diff.Unmodified) != 0 {
					t.Fatalf("Expected 0 Unmodified repos. got %d", len(diff.Added))
				}
			}
		} else {
			for i := 0; i < len(services)*10; i++ {
				diff := <-syncer.Synced

				if len(diff.Added) != 1 {
					t.Fatalf("Expected 1 Added repos. got %d", len(diff.Added))
				}
				if len(diff.Deleted) != 0 {
					t.Fatalf("Expected 0 Deleted repos. got %d", len(diff.Added))
				}
				if len(diff.Modified) != 0 {
					t.Fatalf("Expected 0 Modified repos. got %d", len(diff.Added))
				}
				if len(diff.Unmodified) != 0 {
					t.Fatalf("Expected 0 Unmodified repos. got %d", len(diff.Added))
				}
			}
		}

		var jobsCompleted int
		for i := 0; i < 10; i++ {
			if err := store.QueryRow(ctx, sqlf.Sprintf("SELECT COUNT(*) FROM external_service_sync_jobs where state = 'completed'")).Scan(&jobsCompleted); err != nil {
				t.Fatal(err)
			}
			if jobsCompleted == len(services) {
				break
			}
			// We need to give the worker package time to create the jobs
			time.Sleep(10 * time.Millisecond)
		}

		// Cancel context and the run loop should stop
		cancel()
		err := <-done
		if err != nil && err != context.Canceled {
			t.Fatal(err)
		}
	}
}

func testBatchOrphanedRepo(store *repos.Store) func(*testing.T) {
	return testOrphanedRepo(store, false)
}

func testStreamingOrphanedRepo(store *repos.Store) func(*testing.T) {
	return testOrphanedRepo(store, true)
}

func testOrphanedRepo(store *repos.Store, streaming bool) func(*testing.T) {
	return func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := time.Now()

		svc1 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test1",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		svc2 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test2",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// setup services
		if err := store.ExternalServiceStore.Upsert(ctx, svc1, svc2); err != nil {
			t.Fatal(err)
		}

		githubRepo := &types.Repo{
			Name:     "github.com/org/foo",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-12345",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}

		// Add two services, both pointing at the same repo

		// Sync first service
		syncer := &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc1, nil, githubRepo)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Sync second service
		syncer.Sourcer = func(services ...*types.ExternalService) (repos.Sources, error) {
			s := repos.NewFakeSource(svc2, nil, githubRepo)
			return repos.Sources{s}, nil
		}
		if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Confirm that there are two relationships
		assertSourceCount(ctx, t, store, 2)

		// We should have no deleted repos
		assertDeletedRepoCount(ctx, t, store, 0)

		// Remove the repo from one service and sync again
		syncer.Sourcer = func(services ...*types.ExternalService) (repos.Sources, error) {
			s := repos.NewFakeSource(svc1, nil)
			return repos.Sources{s}, nil
		}
		if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Confirm that the repository hasn't been deleted
		rs, err := store.RepoStore.List(ctx, database.ReposListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(rs) != 1 {
			t.Fatalf("Expected 1 repo, got %d", len(rs))
		}

		// Confirm that there is one relationship
		assertSourceCount(ctx, t, store, 1)

		// We should have no deleted repos
		assertDeletedRepoCount(ctx, t, store, 0)

		// Remove the repo from the second service and sync again
		syncer.Sourcer = func(services ...*types.ExternalService) (repos.Sources, error) {
			s := repos.NewFakeSource(svc2, nil)
			return repos.Sources{s}, nil
		}
		if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Confirm that there no relationships
		assertSourceCount(ctx, t, store, 0)

		// We should have one deleted repo
		assertDeletedRepoCount(ctx, t, store, 1)
	}
}

func testConflictingBatchSyncers(store *repos.Store) func(*testing.T) {
	return testConflictingSyncers(store, false)
}

func testConflictingStreamingSyncers(store *repos.Store) func(*testing.T) {
	return testConflictingSyncers(store, true)
}

func testConflictingSyncers(store *repos.Store, streaming bool) func(*testing.T) {
	return func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := time.Now()

		svc1 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test1",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		svc2 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test2",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// setup services
		if err := store.ExternalServiceStore.Upsert(ctx, svc1, svc2); err != nil {
			t.Fatal(err)
		}

		githubRepo := &types.Repo{
			Name:     "github.com/org/foo",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-12345",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}

		// Add two services, both pointing at the same repo

		// Sync first service
		syncer := &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc1, nil, githubRepo)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Sync second service
		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc2, nil, githubRepo)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Confirm that there are two relationships
		assertSourceCount(ctx, t, store, 2)

		fromDB, err := store.RepoStore.List(ctx, database.ReposListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(fromDB) != 1 {
			t.Fatalf("Expected 1 repo, got %d", len(fromDB))
		}
		beforeUpdate := fromDB[0]
		if beforeUpdate.Description != "" {
			t.Fatalf("Expected %q, got %q", "", beforeUpdate.Description)
		}

		// Create two transactions
		tx1, err := store.Transact(ctx)
		if err != nil {
			t.Fatal(err)
		}

		tx2, err := store.Transact(ctx)
		if err != nil {
			t.Fatal(err)
		}

		newDescription := "This has changed"
		updatedRepo := githubRepo.With(func(r *types.Repo) {
			r.Description = newDescription
		})

		// Start syncing using tx1
		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc1, nil, updatedRepo)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, tx1, svc1.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		if !streaming {
			errChan := make(chan error)
			upsertCalledCh := make(chan struct{})
			go func() {
				// Start syncing using tx2
				syncer2 := &repos.Syncer{
					Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
						s := repos.NewFakeSource(svc2, nil, updatedRepo.With(func(r *types.Repo) {
							r.Description = newDescription
						}))
						return repos.Sources{s}, nil
					},
					Store: store,
					Now:   time.Now,
				}
				storeCp := *store
				storeCp.Mocks.UpsertRepos = func(ctx context.Context, repos ...*types.Repo) (err error) {
					close(upsertCalledCh)
					return store.UpsertRepos(ctx, repos...)
				}
				err := syncer2.SyncExternalService(ctx, &storeCp, svc2.ID, 10*time.Second)
				errChan <- err
			}()

			<-upsertCalledCh
			tx1.Done(nil)

			err = <-errChan
			if err != nil {
				t.Fatalf("Error in syncer2: %v", err)
			}
			tx2.Done(nil)
		} else {
			syncer2 := &repos.Syncer{
				Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc2, nil, githubRepo.With(func(r *types.Repo) {
						r.Description = newDescription
					}))
					return repos.Sources{s}, nil
				},
				Store:     tx2,
				Synced:    make(chan repos.Diff, 2),
				Now:       time.Now,
				Streaming: true,
			}

			errChan := make(chan error)
			go func() {
				errChan <- syncer2.SyncExternalService(ctx, tx2, svc2.ID, 10*time.Second)
			}()

			tx1.Done(nil)

			if err = <-errChan; err != nil {
				t.Fatalf("syncer2 err: %v", err)
			}

			diff := <-syncer2.Synced
			if have, want := diff.Repos().Names(), []string{string(updatedRepo.Name)}; !cmp.Equal(want, have) {
				t.Fatalf("syncer2 Synced mismatch: (-want, +have): %s", cmp.Diff(want, have))
			}

			tx2.Done(nil)
		}

		fromDB, err = store.RepoStore.List(ctx, database.ReposListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(fromDB) != 1 {
			t.Fatalf("Expected 1 repo, got %d", len(fromDB))
		}
		afterUpdate := fromDB[0]
		if afterUpdate.Description != newDescription {
			t.Fatalf("Expected %q, got %q", newDescription, afterUpdate.Description)
		}
	}
}

func testBatchSyncRepoMaintainsOtherSources(store *repos.Store) func(*testing.T) {
	return testSyncRepoMaintainsOtherSources(store, false)
}

func testStreamingSyncRepoMaintainsOtherSources(store *repos.Store) func(*testing.T) {
	return testSyncRepoMaintainsOtherSources(store, true)
}

// Test that sync repo does not clear out any other repo relationships
func testSyncRepoMaintainsOtherSources(store *repos.Store, streaming bool) func(*testing.T) {
	return func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := time.Now()

		svc1 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test1",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		svc2 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test2",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// setup services
		if err := store.ExternalServiceStore.Upsert(ctx, svc1, svc2); err != nil {
			t.Fatal(err)
		}

		githubRepo := &types.Repo{
			Name:     "github.com/org/foo",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-12345",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}

		// Add two services, both pointing at the same repo

		// Sync first service
		syncer := &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc1, nil, githubRepo)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Sync second service
		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc2, nil, githubRepo)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Confirm that there are two relationships
		assertSourceCount(ctx, t, store, 2)

		// Run syncRepo with only one source
		urn := extsvc.URN(extsvc.KindGitHub, svc1.ID)
		githubRepo.Sources = map[string]*types.SourceInfo{
			urn: {
				ID:       urn,
				CloneURL: "cloneURL",
			},
		}
		if err := syncer.SyncRepo(ctx, store, githubRepo); err != nil {
			t.Fatal(err)
		}

		// We should still have two sources
		assertSourceCount(ctx, t, store, 2)
	}
}

func testBatchUserAddedRepos(store *repos.Store) func(*testing.T) {
	return testUserAddedRepos(store, false)
}

func testStreamingUserAddedRepos(store *repos.Store) func(*testing.T) {
	return testUserAddedRepos(store, true)
}

func testUserAddedRepos(store *repos.Store, streaming bool) func(*testing.T) {
	return func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := time.Now()

		var userID int32
		err := store.QueryRow(ctx, sqlf.Sprintf("INSERT INTO users (username) VALUES ('bbs-admin') RETURNING id")).Scan(&userID)
		if err != nil {
			t.Fatal(err)
		}

		userService := &types.ExternalService{
			Kind:            extsvc.KindGitHub,
			DisplayName:     "Github - User",
			Config:          `{"url": "https://github.com"}`,
			CreatedAt:       now,
			UpdatedAt:       now,
			NamespaceUserID: userID,
		}

		adminService := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Private",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// setup services
		if err := store.ExternalServiceStore.Upsert(ctx, userService, adminService); err != nil {
			t.Fatal(err)
		}

		publicRepo := &types.Repo{
			Name:     "github.com/org/user",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-user",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}

		publicRepo2 := &types.Repo{
			Name:     "github.com/org/user2",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-user2",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}

		privateRepo := &types.Repo{
			Name:     "github.com/org/private",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-private",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
			Private: true,
		}

		// Admin service will sync both repos
		syncer := &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(adminService, nil, publicRepo, privateRepo)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, adminService.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Confirm that there are two relationships
		assertSourceCount(ctx, t, store, 2)

		// Unsync the repo to clean things up
		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(adminService, nil)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, adminService.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Confirm that there are zero relationships
		assertSourceCount(ctx, t, store, 0)

		// By default, user service can only sync public code, even if they have access to private code
		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(userService, nil, publicRepo, privateRepo)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, userService.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Confirm that there is one relationship
		assertSourceCount(ctx, t, store, 1)

		// If the private code feature flag is set, user service can also sync private code
		conf.Mock(&conf.Unified{
			SiteConfiguration: schema.SiteConfiguration{
				ExternalServiceUserMode: "all",
			},
		})

		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(userService, nil, publicRepo, privateRepo)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, userService.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Confirm that there are two relationships
		assertSourceCount(ctx, t, store, 2)
		conf.Mock(nil)

		// If the user has the AllowUserExternalServicePrivate tag, user service can also sync private code
		err = store.Exec(ctx, sqlf.Sprintf("UPDATE users SET tags = %s WHERE id = %s", pq.Array([]string{database.TagAllowUserExternalServicePrivate}), userID))
		if err != nil {
			t.Fatal(err)
		}

		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(userService, nil, publicRepo, privateRepo)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, userService.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Confirm that there are two relationships
		assertSourceCount(ctx, t, store, 2)
		err = store.Exec(ctx, sqlf.Sprintf("UPDATE users SET tags = '{}' WHERE id = %s", userID))
		if err != nil {
			t.Fatal(err)
		}

		// Attempt to add some repos with a per user limit set
		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(userService, nil, publicRepo, publicRepo2)
				return repos.Sources{s}, nil
			},
			Now:                 time.Now,
			Store:               store,
			UserReposMaxPerUser: 1,
			Streaming:           streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, userService.ID, 10*time.Second); err == nil {
			t.Fatal("Expected an error, got none")
		}

		// Attempt to add some repos with a total limit set
		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(userService, nil, publicRepo, publicRepo2)
				return repos.Sources{s}, nil
			},
			Now:                 time.Now,
			Store:               store,
			UserReposMaxPerSite: 1,
			Streaming:           streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, userService.ID, 10*time.Second); err == nil {
			t.Fatal("Expected an error, got none")
		}
	}
}

func testNameOnConflictDiscardOld(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		// Test the case where more than one external service returns the same name for different repos. The names
		// are the same, but the external id are different.

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := time.Now()

		svc1 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test1",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		svc2 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test2",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// setup services
		if err := store.ExternalServiceStore.Upsert(ctx, svc1, svc2); err != nil {
			t.Fatal(err)
		}

		githubRepo1 := &types.Repo{
			Name:     "github.com/org/foo",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-foo",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}

		githubRepo2 := &types.Repo{
			Name:     "github.com/org/foo",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-bar",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}

		// Add two services, one with each repo

		// Sync first service
		syncer := &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc1, nil, githubRepo1)
				return repos.Sources{s}, nil
			},
			Store: store,
			Now:   time.Now,
		}
		if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Sync second service
		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc2, nil, githubRepo2)
				return repos.Sources{s}, nil
			},
			Store: store,
			Now:   time.Now,
		}
		if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// We expect repo2 to be synced since it sorts before repo1 because the ID is alphabetically first
		fromDB, err := store.RepoStore.List(ctx, database.ReposListOptions{
			Names: []string{"github.com/org/foo"},
		})
		if err != nil {
			t.Fatal(err)
		}

		if len(fromDB) != 1 {
			t.Fatalf("Expected 1 repo, have %d", len(fromDB))
		}

		found := fromDB[0]
		expectedID := "foo-external-bar"
		if found.ExternalRepo.ID != expectedID {
			t.Fatalf("Want %q, got %q", expectedID, found.ExternalRepo.ID)
		}
	}
}

func testNameOnConflictDiscardNew(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		// Test the case where more than one external service returns the same name for different repos. The names
		// are the same, but the external id are different.

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := time.Now()

		svc1 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test1",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		svc2 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test2",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// setup services
		if err := store.ExternalServiceStore.Upsert(ctx, svc1, svc2); err != nil {
			t.Fatal(err)
		}

		githubRepo1 := &types.Repo{
			Name:     "github.com/org/foo",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-bar",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}

		githubRepo2 := &types.Repo{
			Name:     "github.com/org/foo",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-foo",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}

		// Add two services, one with each repo

		// Sync first service
		syncer := &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc1, nil, githubRepo1)
				return repos.Sources{s}, nil
			},
			Store: store,
			Now:   time.Now,
		}
		if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Sync second service
		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc2, nil, githubRepo2)
				return repos.Sources{s}, nil
			},
			Store: store,
			Now:   time.Now,
		}
		if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// We expect repo1 to be synced since it sorts before repo2 because the ID is alphabetically first
		fromDB, err := store.RepoStore.List(ctx, database.ReposListOptions{
			Names: []string{"github.com/org/foo"},
		})
		if err != nil {
			t.Fatal(err)
		}

		if len(fromDB) != 1 {
			t.Fatalf("Expected 1 repo, have %d", len(fromDB))
		}

		found := fromDB[0]
		expectedID := "foo-external-bar"
		if found.ExternalRepo.ID != expectedID {
			t.Fatalf("Want %q, got %q", expectedID, found.ExternalRepo.ID)
		}
	}
}

func testBatchNameOnConflictOnRename(store *repos.Store) func(*testing.T) {
	return testNameOnConflictOnRename(store, false)
}

func testStreamingNameOnConflictOnRename(store *repos.Store) func(*testing.T) {
	return testNameOnConflictOnRename(store, true)
}

func testNameOnConflictOnRename(store *repos.Store, streaming bool) func(*testing.T) {
	return func(t *testing.T) {
		// Test the case where more than one external service returns the same name for different repos. The names
		// are the same, but the external id are different.

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := time.Now()

		svc1 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test1",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		svc2 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test2",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// setup services
		if err := store.ExternalServiceStore.Upsert(ctx, svc1, svc2); err != nil {
			t.Fatal(err)
		}

		githubRepo1 := &types.Repo{
			Name:     "github.com/org/foo",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-foo",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}

		githubRepo2 := &types.Repo{
			Name:     "github.com/org/bar",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-bar",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}

		// Add two services, one with each repo

		// Sync first service
		syncer := &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc1, nil, githubRepo1)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Sync second service
		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc2, nil, githubRepo2)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Rename repo1 with the same name as repo2
		renamedRepo1 := githubRepo1.With(func(r *types.Repo) {
			r.Name = githubRepo2.Name
		})

		// Sync first service
		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc1, nil, renamedRepo1)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		fromDB, err := store.RepoStore.List(ctx, database.ReposListOptions{})
		if err != nil {
			t.Fatal(err)
		}

		if len(fromDB) != 1 {
			t.Fatalf("Expected 1 repo, have %d", len(fromDB))
		}

		found := fromDB[0]
		var expectedID string

		if !streaming {
			// We expect repo1 to be synced since it sorts before repo2 because the ID is alphabetically first
			expectedID = "foo-external-bar"
		} else {
			// We expect repo2 to be synced since we always pick the just sourced repo as the winner, deleting the other.
			// If the existing conflicting repo still exists, it'll have a different name (because names are unique in
			// the code host), so it'll get re-created once we sync it later.
			expectedID = "foo-external-foo"
		}

		if found.ExternalRepo.ID != expectedID {
			t.Fatalf("Want %q, got %q", expectedID, found.ExternalRepo.ID)
		}
	}
}

func testBatchDeleteExternalService(store *repos.Store) func(*testing.T) {
	return testDeleteExternalService(store, false)
}

func testStreamingDeleteExternalService(store *repos.Store) func(*testing.T) {
	return testDeleteExternalService(store, true)
}

func testDeleteExternalService(store *repos.Store, streaming bool) func(*testing.T) {
	return func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := time.Now()

		svc1 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test1",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		svc2 := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test2",
			Config:      `{"url": "https://github.com"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// setup services
		if err := store.ExternalServiceStore.Upsert(ctx, svc1, svc2); err != nil {
			t.Fatal(err)
		}

		githubRepo := &types.Repo{
			Name:     "github.com/org/foo",
			Metadata: &github.Repository{},
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "foo-external-12345",
				ServiceID:   "https://github.com/",
				ServiceType: extsvc.TypeGitHub,
			},
		}

		// Add two services, both pointing at the same repo

		// Sync first service
		syncer := &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc1, nil, githubRepo)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Sync second service
		syncer = &repos.Syncer{
			Sourcer: func(services ...*types.ExternalService) (repos.Sources, error) {
				s := repos.NewFakeSource(svc2, nil, githubRepo)
				return repos.Sources{s}, nil
			},
			Store:     store,
			Now:       time.Now,
			Streaming: streaming,
		}
		if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
			t.Fatal(err)
		}

		// Delete the first service
		if err := store.ExternalServiceStore.Delete(ctx, svc1.ID); err != nil {
			t.Fatal(err)
		}

		// Confirm that there is one relationship
		assertSourceCount(ctx, t, store, 1)

		// We should have no deleted repos
		assertDeletedRepoCount(ctx, t, store, 0)

		// Delete the second service
		if err := store.ExternalServiceStore.Delete(ctx, svc2.ID); err != nil {
			t.Fatal(err)
		}

		// Confirm that there no relationships
		assertSourceCount(ctx, t, store, 0)

		// We should have one deleted repo
		assertDeletedRepoCount(ctx, t, store, 1)
	}
}

func assertSourceCount(ctx context.Context, t *testing.T, store *repos.Store, want int) {
	t.Helper()
	var rowCount int
	if err := store.QueryRow(ctx, sqlf.Sprintf("SELECT COUNT(*) FROM external_service_repos")).Scan(&rowCount); err != nil {
		t.Fatal(err)
	}
	if rowCount != want {
		t.Fatalf("Expected %d rows, got %d", want, rowCount)
	}
}

func assertDeletedRepoCount(ctx context.Context, t *testing.T, store *repos.Store, want int) {
	t.Helper()
	var rowCount int
	if err := store.QueryRow(ctx, sqlf.Sprintf("SELECT COUNT(*) FROM repo where deleted_at is not null")).Scan(&rowCount); err != nil {
		t.Fatal(err)
	}
	if rowCount != want {
		t.Fatalf("Expected %d rows, got %d", want, rowCount)
	}
}
