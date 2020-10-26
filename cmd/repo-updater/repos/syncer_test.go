package repos_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gitchander/permutation"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/awscodecommit"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketcloud"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitolite"
)

func testSyncerSyncWithErrors(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		githubService := repos.ExternalService{
			Kind:   extsvc.KindGitHub,
			Config: `{}`,
		}

		if err := store.UpsertExternalServices(ctx, &githubService); err != nil {
			t.Fatal(err)
		}

		for _, tc := range []struct {
			name    string
			sourcer repos.Sourcer
			store   repos.Store
			err     string
		}{
			{
				name:    "sourcer error aborts sync",
				sourcer: repos.NewFakeSourcer(errors.New("boom")),
				store:   store,
				err:     "syncer.sync.sourced: 1 error occurred:\n\t* boom\n\n",
			},
			{
				name:    "store list error aborts sync",
				sourcer: repos.NewFakeSourcer(nil, repos.NewFakeSource(&githubService, nil)),
				store: &storeWithErrors{
					Store:        store,
					ListReposErr: errors.New("boom"),
				},
				err: "syncer.sync.store.list-repos: boom",
			},
			{
				name:    "store upsert error aborts sync",
				sourcer: repos.NewFakeSourcer(nil, repos.NewFakeSource(&githubService, nil)),
				store: &storeWithErrors{
					Store:          store,
					UpsertReposErr: errors.New("booya"),
				},
				err: "syncer.sync.store.upsert-repos: booya",
			},
		} {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				clock := repos.NewFakeClock(time.Now(), time.Second)
				now := clock.Now
				ctx := context.Background()

				syncer := &repos.Syncer{
					Sourcer: tc.sourcer,
					Now:     now,
				}
				err := syncer.SyncExternalService(ctx, tc.store, githubService.ID, time.Millisecond)

				if have, want := err.Error(), tc.err; have != want {
					t.Errorf("have error %q, want %q", have, want)
				}

				if len(syncer.SyncErrors()) != 1 {
					t.Fatal("expected 1 error")
				}

				if have, want := syncer.SyncErrors(), tc.err; have[0].Error() != want {
					t.Errorf("have SyncErrors %q, want %q", have, want)
				}
			})
		}
	}
}

type storeWithErrors struct {
	repos.Store

	ListReposErr   error
	UpsertReposErr error
}

func (s *storeWithErrors) ListRepos(ctx context.Context, args repos.StoreListReposArgs) ([]*repos.Repo, error) {
	if s.ListReposErr != nil {
		return nil, s.ListReposErr
	}
	return s.Store.ListRepos(ctx, args)
}

func (s *storeWithErrors) UpsertRepos(ctx context.Context, repos ...*repos.Repo) error {
	if s.UpsertReposErr != nil {
		return s.UpsertReposErr
	}
	return s.Store.UpsertRepos(ctx, repos...)
}

func testSyncerSync(t *testing.T, s repos.Store) func(*testing.T) {
	servicesPerKind := createExternalServices(t, s)

	githubService := servicesPerKind[extsvc.KindGitHub]

	githubRepo := (&repos.Repo{
		Name:     "github.com/org/foo",
		Metadata: &github.Repository{},
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "foo-external-12345",
			ServiceID:   "https://github.com/",
			ServiceType: extsvc.TypeGitHub,
		},
	}).With(
		repos.Opt.RepoSources(githubService.URN()),
	)

	gitlabService := servicesPerKind[extsvc.KindGitLab]

	gitlabRepo := (&repos.Repo{
		Name:     "gitlab.com/org/foo",
		Metadata: &gitlab.Project{},
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "12345",
			ServiceID:   "https://gitlab.com/",
			ServiceType: extsvc.TypeGitLab,
		},
	}).With(
		repos.Opt.RepoSources(gitlabService.URN()),
	)

	bitbucketServerService := servicesPerKind[extsvc.KindBitbucketServer]

	bitbucketServerRepo := (&repos.Repo{
		Name:     "bitbucketserver.mycorp.com/org/foo",
		Metadata: &bitbucketserver.Repo{},
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "23456",
			ServiceID:   "https://bitbucketserver.mycorp.com/",
			ServiceType: "bitbucketServer",
		},
	}).With(
		repos.Opt.RepoSources(bitbucketServerService.URN()),
	)

	awsCodeCommitService := servicesPerKind[extsvc.KindAWSCodeCommit]

	awsCodeCommitRepo := (&repos.Repo{
		Name:     "git-codecommit.us-west-1.amazonaws.com/stripe-go",
		Metadata: &awscodecommit.Repository{},
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "f001337a-3450-46fd-b7d2-650c0EXAMPLE",
			ServiceID:   "arn:aws:codecommit:us-west-1:999999999999:",
			ServiceType: extsvc.TypeAWSCodeCommit,
		},
	}).With(
		repos.Opt.RepoSources(awsCodeCommitService.URN()),
	)

	otherService := servicesPerKind[extsvc.KindOther]

	otherRepo := (&repos.Repo{
		Name: "git-host.com/org/foo",
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "git-host.com/org/foo",
			ServiceID:   "https://git-host.com/",
			ServiceType: extsvc.TypeOther,
		},
	}).With(
		repos.Opt.RepoSources(otherService.URN()),
	)

	gitoliteService := servicesPerKind[extsvc.KindGitolite]

	gitoliteRepo := (&repos.Repo{
		Name:     "gitolite.mycorp.com/foo",
		Metadata: &gitolite.Repo{},
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "foo",
			ServiceID:   "git@gitolite.mycorp.com",
			ServiceType: extsvc.TypeGitolite,
		},
	}).With(
		repos.Opt.RepoSources(gitoliteService.URN()),
	)

	bitbucketCloudService := servicesPerKind[extsvc.KindBitbucketCloud]

	bitbucketCloudRepo := (&repos.Repo{
		Name:     "bitbucket.org/team/foo",
		Metadata: &bitbucketcloud.Repo{},
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "{e164a64c-bd73-4a40-b447-d71b43f328a8}",
			ServiceID:   "https://bitbucket.org/",
			ServiceType: extsvc.TypeBitbucketCloud,
		},
	}).With(
		repos.Opt.RepoSources(bitbucketCloudService.URN()),
	)

	clock := repos.NewFakeClock(time.Now(), 0)

	svcdup := repos.ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "Github2 - Test",
		Config:      `{"url": "https://github.com"}`,
		CreatedAt:   clock.Now(),
		UpdatedAt:   clock.Now(),
	}

	// create a few external services
	if err := s.UpsertExternalServices(context.Background(), &svcdup); err != nil {
		t.Fatalf("failed to insert external services: %v", err)
	}

	var services []repos.ExternalService
	for _, svc := range servicesPerKind {
		services = append(services, *svc)
	}

	type testCase struct {
		name    string
		sourcer repos.Sourcer
		store   repos.Store
		stored  repos.Repos
		svcs    []*repos.ExternalService
		ctx     context.Context
		now     func() time.Time
		diff    repos.Diff
		err     string
	}

	var testCases []testCase
	for _, tc := range []struct {
		repo *repos.Repo
		svc  *repos.ExternalService
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
				name: tc.repo.Name + "/new repo",
				sourcer: repos.NewFakeSourcer(nil,
					repos.NewFakeSource(tc.svc.Clone(), nil, tc.repo.Clone()),
				),
				store:  s,
				stored: repos.Repos{},
				now:    clock.Now,
				diff: repos.Diff{Added: repos.Repos{tc.repo.With(
					repos.Opt.RepoCreatedAt(clock.Time(1)),
					repos.Opt.RepoSources(tc.svc.Clone().URN()),
				)}},
				svcs: []*repos.ExternalService{tc.svc},
				err:  "<nil>",
			},
			testCase{
				name: tc.repo.Name + "/new repo sources",
				sourcer: repos.NewFakeSourcer(nil,
					repos.NewFakeSource(tc.svc.Clone(), nil, tc.repo.Clone()),
					repos.NewFakeSource(svcdup.Clone(), nil, tc.repo.Clone()),
				),
				store:  s,
				stored: repos.Repos{tc.repo.Clone()},
				now:    clock.Now,
				diff: repos.Diff{Modified: repos.Repos{tc.repo.With(
					repos.Opt.RepoModifiedAt(clock.Time(1)),
					repos.Opt.RepoSources(tc.svc.URN(), svcdup.URN()),
				)}},
				svcs: []*repos.ExternalService{tc.svc},
				err:  "<nil>",
			},
			testCase{
				// It's expected that there could be multiple stored sources but only one will ever be returned
				// by the code host as it can't know about others.
				name: tc.repo.Name + "/source already stored",
				sourcer: repos.NewFakeSourcer(nil,
					repos.NewFakeSource(tc.svc.Clone(), nil, tc.repo.Clone()),
				),
				store: s,
				stored: repos.Repos{tc.repo.With(
					repos.Opt.RepoSources(tc.svc.URN(), svcdup.URN()),
				)},
				now: clock.Now,
				diff: repos.Diff{Unmodified: repos.Repos{tc.repo.With(
					repos.Opt.RepoSources(tc.svc.URN(), svcdup.URN()),
				)}},
				svcs: []*repos.ExternalService{tc.svc},
				err:  "<nil>",
			},
			testCase{
				name:    tc.repo.Name + "/deleted ALL repo sources",
				sourcer: repos.NewFakeSourcer(nil),
				store:   s,
				stored: repos.Repos{tc.repo.With(
					repos.Opt.RepoSources(tc.svc.URN(), svcdup.URN()),
				)},
				now: clock.Now,
				diff: repos.Diff{Deleted: repos.Repos{tc.repo.With(
					repos.Opt.RepoDeletedAt(clock.Time(1)),
				)}},
				svcs: []*repos.ExternalService{tc.svc, &svcdup},
				err:  "<nil>",
			},
			testCase{
				name:    tc.repo.Name + "/renamed repo is detected via external_id",
				sourcer: repos.NewFakeSourcer(nil, repos.NewFakeSource(tc.svc.Clone(), nil, tc.repo.Clone())),
				store:   s,
				stored: repos.Repos{tc.repo.With(func(r *repos.Repo) {
					r.Name = "old-name"
				})},
				now: clock.Now,
				diff: repos.Diff{Modified: repos.Repos{
					tc.repo.With(
						repos.Opt.RepoModifiedAt(clock.Time(1))),
				}},
				svcs: []*repos.ExternalService{tc.svc},
				err:  "<nil>",
			},
			testCase{
				name: tc.repo.Name + "/repo got renamed to another repo that gets deleted",
				sourcer: repos.NewFakeSourcer(nil,
					repos.NewFakeSource(tc.svc.Clone(), nil,
						tc.repo.With(func(r *repos.Repo) { r.ExternalRepo.ID = "another-id" }),
					),
				),
				store: s,
				stored: repos.Repos{
					tc.repo.Clone(),
					tc.repo.With(func(r *repos.Repo) {
						r.Name = "another-repo"
						r.ExternalRepo.ID = "another-id"
					}),
				},
				now: clock.Now,
				diff: repos.Diff{
					Deleted: repos.Repos{
						tc.repo.With(func(r *repos.Repo) {
							r.Sources = map[string]*repos.SourceInfo{}
							r.DeletedAt = clock.Time(0)
							r.UpdatedAt = clock.Time(0)
						}),
					},
					Modified: repos.Repos{
						tc.repo.With(
							repos.Opt.RepoModifiedAt(clock.Time(1)),
							func(r *repos.Repo) { r.ExternalRepo.ID = "another-id" },
						),
					},
				},
				svcs: []*repos.ExternalService{tc.svc},
				err:  "<nil>",
			},
			testCase{
				name: tc.repo.Name + "/repo inserted with same name as another repo that gets deleted",
				sourcer: repos.NewFakeSourcer(nil,
					repos.NewFakeSource(tc.svc.Clone(), nil,
						tc.repo,
					),
				),
				store: s,
				stored: repos.Repos{
					tc.repo.With(repos.Opt.RepoExternalID("another-id")),
				},
				now: clock.Now,
				diff: repos.Diff{
					Added: repos.Repos{
						tc.repo.With(
							repos.Opt.RepoCreatedAt(clock.Time(1)),
							repos.Opt.RepoModifiedAt(clock.Time(1)),
						),
					},
					Deleted: repos.Repos{
						tc.repo.With(func(r *repos.Repo) {
							r.ExternalRepo.ID = "another-id"
							r.Sources = map[string]*repos.SourceInfo{}
							r.DeletedAt = clock.Time(0)
							r.UpdatedAt = clock.Time(0)
						}),
					},
				},
				svcs: []*repos.ExternalService{tc.svc},
				err:  "<nil>",
			},
			testCase{
				name: tc.repo.Name + "/repo inserted with same name as repo without id",
				sourcer: repos.NewFakeSourcer(nil,
					repos.NewFakeSource(tc.svc.Clone(), nil,
						tc.repo,
					),
				),
				store: s,
				stored: repos.Repos{
					tc.repo.With(repos.Opt.RepoName("old-name")),  // same external id as sourced
					tc.repo.With(repos.Opt.RepoExternalID("bar")), // same name as sourced
				}.With(repos.Opt.RepoCreatedAt(clock.Time(1))),
				now: clock.Now,
				diff: repos.Diff{
					Modified: repos.Repos{
						tc.repo.With(
							repos.Opt.RepoCreatedAt(clock.Time(1)),
							repos.Opt.RepoModifiedAt(clock.Time(1)),
						),
					},
					Deleted: repos.Repos{
						tc.repo.With(func(r *repos.Repo) {
							r.ExternalRepo.ID = ""
							r.Sources = map[string]*repos.SourceInfo{}
							r.DeletedAt = clock.Time(0)
							r.UpdatedAt = clock.Time(0)
							r.CreatedAt = clock.Time(0)
						}),
					},
				},
				svcs: []*repos.ExternalService{tc.svc},
				err:  "<nil>",
			},
			testCase{
				name:    tc.repo.Name + "/renamed repo which was deleted is detected and added",
				sourcer: repos.NewFakeSourcer(nil, repos.NewFakeSource(tc.svc.Clone(), nil, tc.repo.Clone())),
				store:   s,
				stored: repos.Repos{tc.repo.With(func(r *repos.Repo) {
					r.Sources = map[string]*repos.SourceInfo{}
					r.Name = "old-name"
					r.DeletedAt = clock.Time(0)
				})},
				now: clock.Now,
				diff: repos.Diff{Added: repos.Repos{
					tc.repo.With(
						repos.Opt.RepoCreatedAt(clock.Time(1))),
				}},
				svcs: []*repos.ExternalService{tc.svc},
				err:  "<nil>",
			},
			testCase{
				name: tc.repo.Name + "/repos have their names swapped",
				sourcer: repos.NewFakeSourcer(nil, repos.NewFakeSource(tc.svc.Clone(), nil,
					tc.repo.With(func(r *repos.Repo) {
						r.Name = "foo"
						r.ExternalRepo.ID = "1"
					}),
					tc.repo.With(func(r *repos.Repo) {
						r.Name = "bar"
						r.ExternalRepo.ID = "2"
					}),
				)),
				now:   clock.Now,
				store: s,
				stored: repos.Repos{
					tc.repo.With(func(r *repos.Repo) {
						r.Name = "bar"
						r.ExternalRepo.ID = "1"
					}),
					tc.repo.With(func(r *repos.Repo) {
						r.Name = "foo"
						r.ExternalRepo.ID = "2"
					}),
				},
				diff: repos.Diff{
					Modified: repos.Repos{
						tc.repo.With(func(r *repos.Repo) {
							r.Name = "foo"
							r.ExternalRepo.ID = "1"
							r.UpdatedAt = clock.Time(0)
						}),
						tc.repo.With(func(r *repos.Repo) {
							r.Name = "bar"
							r.ExternalRepo.ID = "2"
							r.UpdatedAt = clock.Time(0)
						}),
					},
				},
				svcs: []*repos.ExternalService{tc.svc},
				err:  "<nil>",
			},
			testCase{
				name: tc.repo.Name + "/case insensitive name",
				sourcer: repos.NewFakeSourcer(nil, repos.NewFakeSource(tc.svc.Clone(), nil,
					tc.repo.Clone(),
					tc.repo.With(repos.Opt.RepoName(strings.ToUpper(tc.repo.Name))),
				)),
				store:  s,
				stored: repos.Repos{tc.repo.With(repos.Opt.RepoName(strings.ToUpper(tc.repo.Name)))},
				now:    clock.Now,
				diff:   repos.Diff{Modified: repos.Repos{tc.repo.With(repos.Opt.RepoModifiedAt(clock.Time(0)))}},
				svcs:   []*repos.ExternalService{tc.svc},
				err:    "<nil>",
			},
			func() testCase {
				var update interface{}
				typ, ok := extsvc.ParseServiceType(tc.repo.ExternalRepo.ServiceType)
				if !ok {
					panic(fmt.Sprintf("test must be extended with new external service kind: %q", strings.ToLower(tc.repo.ExternalRepo.ServiceType)))
				}
				switch typ {
				case extsvc.TypeGitHub:
					update = &github.Repository{
						IsArchived:       true,
						ViewerPermission: "ADMIN",
					}
				case extsvc.TypeGitLab:
					update = &gitlab.Project{Archived: true}
				case extsvc.TypeBitbucketServer:
					update = &bitbucketserver.Repo{Public: true}
				case extsvc.TypeBitbucketCloud:
					update = &bitbucketcloud.Repo{IsPrivate: true}
				case extsvc.TypeAWSCodeCommit:
					update = &awscodecommit.Repository{Description: "new description"}
				case extsvc.TypeOther, extsvc.TypeGitolite:
					return testCase{}
				default:
					panic(fmt.Sprintf("test must be extended with new external service kind: %q", strings.ToLower(tc.repo.ExternalRepo.ServiceType)))
				}

				expected := update
				// Special case for GitHub, see Repo.Update method
				if typ == extsvc.TypeGitHub {
					expected = &github.Repository{
						IsArchived: true,
					}
				}

				return testCase{
					name: tc.repo.Name + "/metadata update",
					sourcer: repos.NewFakeSourcer(nil, repos.NewFakeSource(tc.svc.Clone(), nil,
						tc.repo.With(repos.Opt.RepoModifiedAt(clock.Time(1)),
							repos.Opt.RepoMetadata(update)),
					)),
					store:  s,
					stored: repos.Repos{tc.repo.Clone()},
					now:    clock.Now,
					diff: repos.Diff{Modified: repos.Repos{tc.repo.With(
						repos.Opt.RepoModifiedAt(clock.Time(1)),
						repos.Opt.RepoMetadata(expected),
					)}},
					svcs: []*repos.ExternalService{tc.svc},
					err:  "<nil>",
				}
			}(),
		)
	}

	return func(t *testing.T) {
		t.Helper()

		for _, tc := range testCases {
			if tc.name == "" {
				continue
			}

			tc := tc
			ctx := context.Background()

			t.Run(tc.name, transact(ctx, tc.store, func(t testing.TB, st repos.Store) {
				defer func() {
					if err := recover(); err != nil {
						t.Fatalf("%q panicked: %v", tc.name, err)
					}
				}()

				now := tc.now
				if now == nil {
					clock := repos.NewFakeClock(time.Now(), time.Second)
					now = clock.Now
				}

				ctx := tc.ctx
				if ctx == nil {
					ctx = context.Background()
				}

				if st != nil && len(tc.stored) > 0 {
					cloned := tc.stored.Clone()
					if err := st.InsertRepos(ctx, cloned...); err != nil {
						t.Fatalf("failed to prepare store: %v", err)
					}
				}

				syncer := &repos.Syncer{
					Sourcer: tc.sourcer,
					Now:     now,
				}

				for _, svc := range tc.svcs {
					err := syncer.SyncExternalService(ctx, st, svc.ID, time.Millisecond)

					if have, want := fmt.Sprint(err), tc.err; have != want {
						t.Errorf("have error %q, want %q", have, want)
					}

					if err != nil {
						return
					}
				}

				if st != nil {
					var want, have repos.Repos
					want.Concat(tc.diff.Added, tc.diff.Modified, tc.diff.Unmodified)
					have, _ = st.ListRepos(ctx, repos.StoreListReposArgs{})

					want = want.With(repos.Opt.RepoID(0))
					have = have.With(repos.Opt.RepoID(0))
					sort.Sort(want)
					sort.Sort(have)

					repos.Assert.ReposEqual(want...)(t, have)
				}
			}))
		}
	}
}

func testSyncRepo(t *testing.T, s repos.Store) func(*testing.T) {
	clock := repos.NewFakeClock(time.Now(), time.Second)

	servicesPerKind := createExternalServices(t, s)

	repo := &repos.Repo{
		ID:          0, // explicitly make default value for sourced repo
		Name:        "github.com/foo/bar",
		Description: "The description",
		Archived:    false,
		Fork:        false,
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			ServiceType: extsvc.TypeGitHub,
			ServiceID:   "https://github.com/",
		},
		Sources: map[string]*repos.SourceInfo{
			servicesPerKind[extsvc.KindGitHub].URN(): {
				ID:       servicesPerKind[extsvc.KindGitHub].URN(),
				CloneURL: "git@github.com:foo/bar.git",
			},
		},
		Metadata: &github.Repository{
			ID:            "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
			URL:           "github.com/foo/bar",
			DatabaseID:    1234,
			Description:   "The description",
			NameWithOwner: "foo/bar",
		},
	}

	testCases := []struct {
		name    string
		sourced *repos.Repo
		stored  repos.Repos
		assert  repos.ReposAssertion
	}{{
		name:    "insert",
		sourced: repo,
		assert:  repos.Assert.ReposEqual(repo.With(repos.Opt.RepoCreatedAt(clock.Time(2)))),
	}, {
		name:    "update",
		sourced: repo,
		stored:  repos.Repos{repo.With(repos.Opt.RepoCreatedAt(clock.Time(2)))},
		assert: repos.Assert.ReposEqual(repo.With(
			repos.Opt.RepoModifiedAt(clock.Time(2)),
			repos.Opt.RepoCreatedAt(clock.Time(2)))),
	}, {
		name:    "update name",
		sourced: repo,
		stored: repos.Repos{repo.With(
			repos.Opt.RepoName("old/name"),
			repos.Opt.RepoCreatedAt(clock.Time(2)))},
		assert: repos.Assert.ReposEqual(repo.With(
			repos.Opt.RepoModifiedAt(clock.Time(2)),
			repos.Opt.RepoCreatedAt(clock.Time(2)))),
	}, {
		name:    "delete conflicting name",
		sourced: repo,
		stored: repos.Repos{repo.With(
			repos.Opt.RepoExternalID("old id"),
			repos.Opt.RepoCreatedAt(clock.Time(2)))},
		assert: repos.Assert.ReposEqual(repo.With(
			repos.Opt.RepoCreatedAt(clock.Time(2)))),
	}, {
		name:    "rename and delete conflicting name",
		sourced: repo,
		stored: repos.Repos{
			repo.With(
				repos.Opt.RepoExternalID("old id"),
				repos.Opt.RepoCreatedAt(clock.Time(2))),
			repo.With(
				repos.Opt.RepoName("old name"),
				repos.Opt.RepoCreatedAt(clock.Time(2))),
		},
		assert: repos.Assert.ReposEqual(repo.With(
			repos.Opt.RepoCreatedAt(clock.Time(2)))),
	}}

	return func(t *testing.T) {
		t.Helper()

		for _, tc := range testCases {
			if tc.name == "" {
				continue
			}

			tc := tc
			ctx := context.Background()

			t.Run(tc.name, transact(ctx, s, func(t testing.TB, st repos.Store) {
				defer func() {
					if err := recover(); err != nil {
						t.Fatalf("%q panicked: %v", tc.name, err)
					}
				}()

				if len(tc.stored) > 0 {
					if err := st.InsertRepos(ctx, tc.stored.Clone()...); err != nil {
						t.Fatalf("failed to prepare store: %v", err)
					}
				}

				clock := clock
				syncer := &repos.Syncer{
					Now: clock.Now,
				}
				err := syncer.SyncRepo(ctx, st, tc.sourced.Clone())
				if err != nil {
					t.Fatal(err)
				}

				have, err := st.ListRepos(ctx, repos.StoreListReposArgs{})
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
		store  repos.Repos
		source repos.Repos
		diff   repos.Diff
	}

	cases := []testCase{
		{
			name: "empty",
			diff: repos.Diff{},
		},
		{
			name:   "added",
			source: repos.Repos{{ExternalRepo: eid("1")}},
			diff:   repos.Diff{Added: repos.Repos{{ExternalRepo: eid("1")}}},
		},
		{
			name:  "deleted",
			store: repos.Repos{{ExternalRepo: eid("1")}},
			diff:  repos.Diff{Deleted: repos.Repos{{ExternalRepo: eid("1")}}},
		},
		{
			name: "modified",
			store: repos.Repos{
				{ExternalRepo: eid("1"), Name: "foo", Description: "foo"},
				{ExternalRepo: eid("2"), Name: "bar"},
			},
			source: repos.Repos{
				{ExternalRepo: eid("1"), Name: "foo", Description: "bar"},
				{ExternalRepo: eid("2"), Name: "bar", URI: "2"},
			},
			diff: repos.Diff{Modified: repos.Repos{
				{ExternalRepo: eid("1"), Name: "foo", Description: "bar"},
				{ExternalRepo: eid("2"), Name: "bar", URI: "2"},
			}},
		},
		{
			name:   "unmodified",
			store:  repos.Repos{{ExternalRepo: eid("1"), Description: "foo"}},
			source: repos.Repos{{ExternalRepo: eid("1"), Description: "foo"}},
			diff: repos.Diff{Unmodified: repos.Repos{
				{ExternalRepo: eid("1"), Description: "foo"},
			}},
		},
		{
			name: "duplicates in source are merged",
			source: repos.Repos{
				{ExternalRepo: eid("1"), Description: "foo", Sources: map[string]*repos.SourceInfo{
					"a": {ID: "a"},
				}},
				{ExternalRepo: eid("1"), Description: "bar", Sources: map[string]*repos.SourceInfo{
					"b": {ID: "b"},
				}},
			},
			diff: repos.Diff{Added: repos.Repos{
				{ExternalRepo: eid("1"), Description: "bar", Sources: map[string]*repos.SourceInfo{
					"a": {ID: "a"},
					"b": {ID: "b"},
				}},
			}},
		},
		{
			name: "duplicate with a changed name is merged correctly",
			store: repos.Repos{
				{Name: "1", ExternalRepo: eid("1"), Description: "foo"},
			},
			source: repos.Repos{
				{Name: "2", ExternalRepo: eid("1"), Description: "foo"},
			},
			diff: repos.Diff{Modified: repos.Repos{
				{Name: "2", ExternalRepo: eid("1"), Description: "foo"},
			}},
		},
		{
			name: "unmodified preserves stored repo",
			store: repos.Repos{
				{ExternalRepo: eid("1"), Description: "foo", UpdatedAt: now},
			},
			source: repos.Repos{
				{ExternalRepo: eid("1"), Description: "foo"},
			},
			diff: repos.Diff{Unmodified: repos.Repos{
				{ExternalRepo: eid("1"), Description: "foo", UpdatedAt: now},
			}},
		},
	}

	permutedCases := []testCase{
		{
			// Repo renamed and repo created with old name
			name: "renamed repo",
			store: repos.Repos{
				{Name: "1", ExternalRepo: eid("old"), Description: "foo"},
			},
			source: repos.Repos{
				{Name: "2", ExternalRepo: eid("old"), Description: "foo"},
				{Name: "1", ExternalRepo: eid("new"), Description: "bar"},
			},
			diff: repos.Diff{
				Modified: repos.Repos{
					{Name: "2", ExternalRepo: eid("old"), Description: "foo"},
				},
				Added: repos.Repos{
					{Name: "1", ExternalRepo: eid("new"), Description: "bar"},
				},
			},
		},
		{
			name:  "repo renamed to an already deleted repo",
			store: repos.Repos{},
			source: repos.Repos{
				{Name: "a", ExternalRepo: eid("b")},
			},
			diff: repos.Diff{
				Added: repos.Repos{
					{Name: "a", ExternalRepo: eid("b")},
				},
			},
		},
		{
			name: "repo renamed to a repo that gets deleted",
			store: repos.Repos{
				{Name: "a", ExternalRepo: eid("a")},
				{Name: "b", ExternalRepo: eid("b")},
			},
			source: repos.Repos{
				{Name: "a", ExternalRepo: eid("b")},
			},
			diff: repos.Diff{
				Deleted:  repos.Repos{{Name: "a", ExternalRepo: eid("a")}},
				Modified: repos.Repos{{Name: "a", ExternalRepo: eid("b")}},
			},
		},
		{
			name: "swapped repo",
			store: repos.Repos{
				{Name: "foo", ExternalRepo: eid("1"), Description: "foo"},
				{Name: "bar", ExternalRepo: eid("2"), Description: "bar"},
			},
			source: repos.Repos{
				{Name: "bar", ExternalRepo: eid("1"), Description: "bar"},
				{Name: "foo", ExternalRepo: eid("2"), Description: "foo"},
			},
			diff: repos.Diff{
				Modified: repos.Repos{
					{Name: "bar", ExternalRepo: eid("1"), Description: "bar"},
					{Name: "foo", ExternalRepo: eid("2"), Description: "foo"},
				},
			},
		},
		{
			name: "deterministic merging of source",
			source: repos.Repos{
				{Name: "foo", ExternalRepo: eid("1"), Description: "desc1", Sources: map[string]*repos.SourceInfo{"a": nil}},
				{Name: "foo", ExternalRepo: eid("1"), Description: "desc2", Sources: map[string]*repos.SourceInfo{"b": nil}},
			},
			diff: repos.Diff{
				Added: repos.Repos{
					{Name: "foo", ExternalRepo: eid("1"), Description: "desc2", Sources: map[string]*repos.SourceInfo{"a": nil, "b": nil}},
				},
			},
		},
		{
			name: "conflict on case insensitive name",
			source: repos.Repos{
				{Name: "foo", ExternalRepo: eid("1")},
				{Name: "Foo", ExternalRepo: eid("2")},
			},
			diff: repos.Diff{
				Added: repos.Repos{
					{Name: "Foo", ExternalRepo: eid("2")},
				},
			},
		},
		{
			name: "conflict on case insensitive name exists 1",
			store: repos.Repos{
				{Name: "foo", ExternalRepo: eid("1")},
			},
			source: repos.Repos{
				{Name: "foo", ExternalRepo: eid("1")},
				{Name: "Foo", ExternalRepo: eid("2")},
			},
			diff: repos.Diff{
				Added: repos.Repos{
					{Name: "Foo", ExternalRepo: eid("2")},
				},
				Deleted: repos.Repos{
					{Name: "foo", ExternalRepo: eid("1")},
				},
			},
		},
		{
			name: "conflict on case insensitive name exists 2",
			store: repos.Repos{
				{Name: "Foo", ExternalRepo: eid("2")},
			},
			source: repos.Repos{
				{Name: "foo", ExternalRepo: eid("1")},
				{Name: "Foo", ExternalRepo: eid("2")},
			},
			diff: repos.Diff{
				Unmodified: repos.Repos{
					{Name: "Foo", ExternalRepo: eid("2")},
				},
			},
		},
		// 🚨 SECURITY: Tests to ensure we detect repository visibility changes.
		{
			name: "repository visiblity changed",
			store: repos.Repos{
				{Name: "foo", ExternalRepo: eid("1"), Description: "foo", Private: false},
			},
			source: repos.Repos{
				{Name: "foo", ExternalRepo: eid("1"), Description: "foo", Private: true},
			},
			diff: repos.Diff{
				Modified: repos.Repos{
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

func testSyncRun(db *sql.DB) func(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T, store repos.Store) func(t *testing.T) {
		return func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			svc := &repos.ExternalService{
				Config: `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "abc"}`,
				Kind:   extsvc.KindGitHub,
			}

			if err := store.UpsertExternalServices(ctx, svc); err != nil {
				t.Fatal(err)
			}

			mk := func(name string) *repos.Repo {
				return &repos.Repo{
					Name:     name,
					Metadata: &github.Repository{},
					ExternalRepo: api.ExternalRepoSpec{
						ID:          name,
						ServiceID:   "https://github.com",
						ServiceType: svc.Kind,
					},
				}
			}

			// Our test will have 1 initial repo, and discover a new repo on sourcing.
			stored := repos.Repos{mk("initial")}.With(repos.Opt.RepoSources(svc.URN()))
			sourced := repos.Repos{mk("initial"), mk("new")}

			syncer := &repos.Syncer{
				Sourcer:      repos.NewFakeSourcer(nil, repos.NewFakeSource(svc, nil, sourced...)),
				Store:        store,
				Synced:       make(chan repos.Diff),
				SubsetSynced: make(chan repos.Diff),
				Now:          time.Now,
			}

			// Initial repos in store
			if err := store.InsertRepos(ctx, stored...); err != nil {
				t.Fatal(err)
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				err := syncer.Run(ctx, db, store, repos.RunOptions{
					EnqueueInterval: func() time.Duration { return time.Second },
					IsCloud:         false,
					MinSyncInterval: func() time.Duration { return 1 * time.Millisecond },
					DequeueInterval: 1 * time.Millisecond,
				})
				if err != nil && err != context.Canceled {
					t.Fatal(err)
				}
			}()

			// Ignore fields store adds
			ignore := cmpopts.IgnoreFields(repos.Repo{}, "ID", "CreatedAt", "UpdatedAt", "Sources")

			// The first thing sent down Synced is the list of repos in store.
			diff := <-syncer.Synced
			if d := cmp.Diff(repos.Diff{Unmodified: stored}, diff, ignore); d != "" {
				t.Fatalf("initial Synced mismatch (-want +got):\n%s", d)
			}

			// Next up it should find the new repo and send it down SubsetSynced
			diff = <-syncer.SubsetSynced
			if d := cmp.Diff(repos.Diff{Added: repos.Repos{mk("new")}}, diff, ignore); d != "" {
				t.Fatalf("SubsetSynced mismatch (-want +got):\n%s", d)
			}

			// Finally we get the final diff, which will have everything listed as
			// Unmodified since we added when we did SubsetSynced.
			diff = <-syncer.Synced
			if d := cmp.Diff(repos.Diff{Unmodified: sourced}, diff, ignore); d != "" {
				t.Fatalf("final Synced mismatch (-want +got):\n%s", d)
			}

			// We check synced again to test us going around the Run loop 2 times in
			// total.
			diff = <-syncer.Synced
			if d := cmp.Diff(repos.Diff{Unmodified: sourced}, diff, ignore); d != "" {
				t.Fatalf("second final Synced mismatch (-want +got):\n%s", d)
			}

			// Cancel context and the run loop should stop
			cancel()
			<-done
		}
	}
}

func testSyncer(db *sql.DB) func(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T, store repos.Store) func(t *testing.T) {
		return func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			services := mkExternalServices(time.Now())

			githubService := services[0]
			gitlabService := services[1]
			bitbucketCloudService := services[3]

			services = repos.ExternalServices{
				githubService,
				gitlabService,
				bitbucketCloudService,
			}

			// setup services
			if err := store.UpsertExternalServices(ctx, services...); err != nil {
				t.Fatal(err)
			}

			githubRepo := (&repos.Repo{
				Name:     "github.com/org/foo",
				Metadata: &github.Repository{},
				ExternalRepo: api.ExternalRepoSpec{
					ID:          "foo-external-12345",
					ServiceID:   "https://github.com/",
					ServiceType: extsvc.TypeGitHub,
				},
			}).With(
				repos.Opt.RepoSources(githubService.URN()),
			)

			gitlabRepo := (&repos.Repo{
				Name:     "gitlab.com/org/foo",
				Metadata: &gitlab.Project{},
				ExternalRepo: api.ExternalRepoSpec{
					ID:          "12345",
					ServiceID:   "https://gitlab.com/",
					ServiceType: extsvc.TypeGitLab,
				},
			}).With(
				repos.Opt.RepoSources(gitlabService.URN()),
			)

			bitbucketCloudRepo := (&repos.Repo{
				Name:     "bitbucket.org/team/foo",
				Metadata: &bitbucketcloud.Repo{},
				ExternalRepo: api.ExternalRepoSpec{
					ID:          "{e164a64c-bd73-4a40-b447-d71b43f328a8}",
					ServiceID:   "https://bitbucket.org/",
					ServiceType: extsvc.TypeBitbucketCloud,
				},
			}).With(
				repos.Opt.RepoSources(bitbucketCloudService.URN()),
			)

			removeSources := func(r *repos.Repo) {
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
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					if len(services) > 1 {
						t.Fatalf("Expected 1 service, got %d", len(services))
					}
					s, ok := sourcers[services[0].ID]
					if !ok {
						t.Fatalf("sourcer not found: %d", services[0].ID)
					}
					return repos.Sources{s}, nil
				},
				Store:  store,
				Synced: make(chan repos.Diff),
				Now:    time.Now,
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				err := syncer.Run(ctx, db, store, repos.RunOptions{
					EnqueueInterval: func() time.Duration { return time.Second },
					IsCloud:         false,
					MinSyncInterval: func() time.Duration { return 1 * time.Minute },
					DequeueInterval: 1 * time.Millisecond,
				})
				if err != nil && err != context.Canceled {
					t.Fatal(err)
				}
			}()

			// Ignore fields store adds
			ignore := cmpopts.IgnoreFields(repos.Repo{}, "ID", "CreatedAt", "UpdatedAt", "Sources")

			// The first thing sent down Synced is an empty list of repos in store.
			diff := <-syncer.Synced
			if d := cmp.Diff(repos.Diff{}, diff, ignore); d != "" {
				t.Fatalf("initial Synced mismatch (-want +got):\n%s", d)
			}

			// it should add a job for all external services
			var jobCount int
			for i := 0; i < 10; i++ {
				if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM external_service_sync_jobs").Scan(&jobCount); err != nil {
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

			var jobsCompleted int
			for i := 0; i < 10; i++ {
				if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM external_service_sync_jobs where state = 'completed'").Scan(&jobsCompleted); err != nil {
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
			<-done
		}
	}
}

func testOrphanedRepo(db *sql.DB) func(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T, store repos.Store) func(t *testing.T) {
		return func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			now := time.Now()

			svc1 := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test1",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			svc2 := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test2",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			// setup services
			if err := store.UpsertExternalServices(ctx, svc1, svc2); err != nil {
				t.Fatal(err)
			}

			githubRepo := &repos.Repo{
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
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc1, nil, githubRepo)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Sync second service
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc2, nil, githubRepo)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Confirm that there are two relationships
			assertSourceCount(ctx, t, db, 2)

			// We should have no deleted repos
			assertDeletedRepoCount(ctx, t, db, 0)

			// Remove the repo from one service and sync again
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc1, nil)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Confirm that the repository hasn't been deleted
			rs, err := store.ListRepos(ctx, repos.StoreListReposArgs{})
			if err != nil {
				t.Fatal(err)
			}
			if len(rs) != 1 {
				t.Fatalf("Expected 1 repo, got %d", len(rs))
			}

			// Confirm that there is one relationship
			assertSourceCount(ctx, t, db, 1)

			// We should have no deleted repos
			assertDeletedRepoCount(ctx, t, db, 0)

			// Remove the repo from the second service and sync again
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc2, nil)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Confirm that there no relationships
			assertSourceCount(ctx, t, db, 0)

			// We should have one deleted repo
			assertDeletedRepoCount(ctx, t, db, 1)
		}
	}
}

// storeWrapper executes arbitrary code before store methods.
type storeWrapper struct {
	repos.Store

	onUpsertRepos func()
}

func (s *storeWrapper) UpsertRepos(ctx context.Context, rs ...*repos.Repo) error {
	if s.onUpsertRepos != nil {
		s.onUpsertRepos()
	}

	return s.Store.UpsertRepos(ctx, rs...)
}

func testConflictingSyncers(db *sql.DB) func(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T, store repos.Store) func(t *testing.T) {
		return func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			now := time.Now()

			svc1 := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test1",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			svc2 := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test2",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			// setup services
			if err := store.UpsertExternalServices(ctx, svc1, svc2); err != nil {
				t.Fatal(err)
			}

			githubRepo := &repos.Repo{
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
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc1, nil, githubRepo)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Sync second service
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc2, nil, githubRepo)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Confirm that there are two relationships
			assertSourceCount(ctx, t, db, 2)

			fromDB, err := store.ListRepos(ctx, repos.StoreListReposArgs{})
			if len(fromDB) != 1 {
				t.Fatalf("Expected 1 repo, got %d", len(fromDB))
			}
			beforeUpdate := fromDB[0]
			if beforeUpdate.Description != "" {
				t.Fatalf("Expected %q, got %q", "", beforeUpdate.Description)
			}

			// Create two transactions
			txStore := store.(repos.Transactor)
			tx1, err := txStore.Transact(ctx)
			if err != nil {
				t.Fatal(err)
			}

			tx2, err := txStore.Transact(ctx)
			if err != nil {
				t.Fatal(err)
			}

			newDescription := "This has changed"
			updatedRepo := githubRepo.With(func(r *repos.Repo) {
				r.Description = newDescription
			})

			// Start syncing using tx1
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc1, nil, updatedRepo)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, tx1, svc1.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			errChan := make(chan error)
			upsertCalledCh := make(chan struct{})
			go func() {
				// Start syncing using tx2
				syncer2 := &repos.Syncer{
					Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
						s := repos.NewFakeSource(svc2, nil, updatedRepo.With(func(r *repos.Repo) {
							r.Description = newDescription
						}))
						return repos.Sources{s}, nil
					},
					Now: time.Now,
				}
				err := syncer2.SyncExternalService(ctx, &storeWrapper{Store: tx2, onUpsertRepos: func() {
					close(upsertCalledCh)
				}}, svc2.ID, 10*time.Second)
				errChan <- err
			}()

			<-upsertCalledCh
			tx1.Done(nil)

			err = <-errChan
			if err != nil {
				t.Fatalf("Error in syncer2: %v", err)
			}
			tx2.Done(nil)

			fromDB, err = store.ListRepos(ctx, repos.StoreListReposArgs{})
			if len(fromDB) != 1 {
				t.Fatalf("Expected 1 repo, got %d", len(fromDB))
			}
			afterUpdate := fromDB[0]
			if afterUpdate.Description != newDescription {
				t.Fatalf("Expected %q, got %q", newDescription, afterUpdate.Description)
			}
		}
	}
}

func testUserAddedRepos(db *sql.DB, userID int32) func(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T, store repos.Store) func(t *testing.T) {
		return func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			now := time.Now()

			userService := &repos.ExternalService{
				Kind:            extsvc.KindGitHub,
				DisplayName:     "Github - User",
				Config:          `{"url": "https://github.com"}`,
				CreatedAt:       now,
				UpdatedAt:       now,
				NamespaceUserID: userID,
			}

			adminService := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Private",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			// setup services
			if err := store.UpsertExternalServices(ctx, userService, adminService); err != nil {
				t.Fatal(err)
			}

			publicRepo := &repos.Repo{
				Name:     "github.com/org/user",
				Metadata: &github.Repository{},
				ExternalRepo: api.ExternalRepoSpec{
					ID:          "foo-external-user",
					ServiceID:   "https://github.com/",
					ServiceType: extsvc.TypeGitHub,
				},
			}

			publicRepo2 := &repos.Repo{
				Name:     "github.com/org/user2",
				Metadata: &github.Repository{},
				ExternalRepo: api.ExternalRepoSpec{
					ID:          "foo-external-user2",
					ServiceID:   "https://github.com/",
					ServiceType: extsvc.TypeGitHub,
				},
			}

			privateRepo := &repos.Repo{
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
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(adminService, nil, publicRepo, privateRepo)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, adminService.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Confirm that there are two relationships
			assertSourceCount(ctx, t, db, 2)

			// Unsync the repo to clean things up
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(adminService, nil)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, adminService.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Confirm that there are zero relationships
			assertSourceCount(ctx, t, db, 0)

			// User service can only sync public code, even if they have access to private code
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(userService, nil, publicRepo, privateRepo)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, userService.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Confirm that there is one relationship
			assertSourceCount(ctx, t, db, 1)

			// Attempt to add some repos with a per user limit set
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(userService, nil, publicRepo, publicRepo2)
					return repos.Sources{s}, nil
				},
				Now:                 time.Now,
				UserReposMaxPerUser: 1,
			}
			if err := syncer.SyncExternalService(ctx, store, userService.ID, 10*time.Second); err == nil {
				t.Fatal("Expected an error, got none")
			}

			// Attempt to add some repos with a total limit set
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(userService, nil, publicRepo, publicRepo2)
					return repos.Sources{s}, nil
				},
				Now:                 time.Now,
				UserReposMaxPerSite: 1,
			}
			if err := syncer.SyncExternalService(ctx, store, userService.ID, 10*time.Second); err == nil {
				t.Fatal("Expected an error, got none")
			}
		}
	}
}

func testNameOnConflictDiscardOld(db *sql.DB) func(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T, store repos.Store) func(t *testing.T) {
		return func(t *testing.T) {
			// Test the case where more than one external service returns the same name for different repos. The names
			// are the same, but the external id are different.

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			now := time.Now()

			svc1 := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test1",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			svc2 := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test2",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			// setup services
			if err := store.UpsertExternalServices(ctx, svc1, svc2); err != nil {
				t.Fatal(err)
			}

			githubRepo1 := &repos.Repo{
				Name:     "github.com/org/foo",
				Metadata: &github.Repository{},
				ExternalRepo: api.ExternalRepoSpec{
					ID:          "foo-external-foo",
					ServiceID:   "https://github.com/",
					ServiceType: extsvc.TypeGitHub,
				},
			}

			githubRepo2 := &repos.Repo{
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
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc1, nil, githubRepo1)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Sync second service
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc2, nil, githubRepo2)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// We expect repo2 to be synced since it sorts before repo1 because the ID is alphabetically first
			fromDB, err := store.ListRepos(ctx, repos.StoreListReposArgs{
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
}

func testNameOnConflictDiscardNew(db *sql.DB) func(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T, store repos.Store) func(t *testing.T) {
		return func(t *testing.T) {
			// Test the case where more than one external service returns the same name for different repos. The names
			// are the same, but the external id are different.

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			now := time.Now()

			svc1 := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test1",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			svc2 := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test2",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			// setup services
			if err := store.UpsertExternalServices(ctx, svc1, svc2); err != nil {
				t.Fatal(err)
			}

			githubRepo1 := &repos.Repo{
				Name:     "github.com/org/foo",
				Metadata: &github.Repository{},
				ExternalRepo: api.ExternalRepoSpec{
					ID:          "foo-external-bar",
					ServiceID:   "https://github.com/",
					ServiceType: extsvc.TypeGitHub,
				},
			}

			githubRepo2 := &repos.Repo{
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
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc1, nil, githubRepo1)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Sync second service
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc2, nil, githubRepo2)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// We expect repo1 to be synced since it sorts before repo2 because the ID is alphabetically first
			fromDB, err := store.ListRepos(ctx, repos.StoreListReposArgs{
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
}

func testNameOnConflictOnRename(db *sql.DB) func(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T, store repos.Store) func(t *testing.T) {
		return func(t *testing.T) {
			// Test the case where more than one external service returns the same name for different repos. The names
			// are the same, but the external id are different.

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			now := time.Now()

			svc1 := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test1",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			svc2 := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test2",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			// setup services
			if err := store.UpsertExternalServices(ctx, svc1, svc2); err != nil {
				t.Fatal(err)
			}

			githubRepo1 := &repos.Repo{
				Name:     "github.com/org/foo",
				Metadata: &github.Repository{},
				ExternalRepo: api.ExternalRepoSpec{
					ID:          "foo-external-foo",
					ServiceID:   "https://github.com/",
					ServiceType: extsvc.TypeGitHub,
				},
			}

			githubRepo2 := &repos.Repo{
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
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc1, nil, githubRepo1)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Sync second service
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc2, nil, githubRepo2)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Rename repo1 with the same name as repo2
			renamedRepo1 := githubRepo1.With(func(r *repos.Repo) {
				r.Name = githubRepo2.Name
			})

			// Sync first service
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc1, nil, renamedRepo1)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// We expect repo1 to be synced since it sorts before repo2 because the ID is alphabetically first
			fromDB, err := store.ListRepos(ctx, repos.StoreListReposArgs{})
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
}
func testDeleteExternalService(db *sql.DB) func(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T, store repos.Store) func(t *testing.T) {
		return func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			now := time.Now()

			svc1 := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test1",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			svc2 := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "Github - Test2",
				Config:      `{"url": "https://github.com"}`,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			// setup services
			if err := store.UpsertExternalServices(ctx, svc1, svc2); err != nil {
				t.Fatal(err)
			}

			githubRepo := &repos.Repo{
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
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc1, nil, githubRepo)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc1.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Sync second service
			syncer = &repos.Syncer{
				Sourcer: func(services ...*repos.ExternalService) (repos.Sources, error) {
					s := repos.NewFakeSource(svc2, nil, githubRepo)
					return repos.Sources{s}, nil
				},
				Now: time.Now,
			}
			if err := syncer.SyncExternalService(ctx, store, svc2.ID, 10*time.Second); err != nil {
				t.Fatal(err)
			}

			// Delete the first service
			svc1.DeletedAt = now
			if err := store.UpsertExternalServices(ctx, svc1); err != nil {
				t.Fatal(err)
			}

			// Confirm that there is one relationship
			assertSourceCount(ctx, t, db, 1)

			// We should have no deleted repos
			assertDeletedRepoCount(ctx, t, db, 0)

			// Delete the second service
			svc2.DeletedAt = now
			if err := store.UpsertExternalServices(ctx, svc2); err != nil {
				t.Fatal(err)
			}

			// Confirm that there no relationships
			assertSourceCount(ctx, t, db, 0)

			// We should have one deleted repo
			assertDeletedRepoCount(ctx, t, db, 1)
		}
	}
}

func assertSourceCount(ctx context.Context, t *testing.T, db *sql.DB, want int) {
	t.Helper()
	var rowCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM external_service_repos").Scan(&rowCount); err != nil {
		t.Fatal(err)
	}
	if rowCount != want {
		t.Fatalf("Expected %d rows, got %d", want, rowCount)
	}
}

func assertDeletedRepoCount(ctx context.Context, t *testing.T, db *sql.DB, want int) {
	t.Helper()
	var rowCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM repo where deleted_at is not null").Scan(&rowCount); err != nil {
		t.Fatal(err)
	}
	if rowCount != want {
		t.Fatalf("Expected %d rows, got %d", want, rowCount)
	}
}
