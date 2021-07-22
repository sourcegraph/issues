package repos_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)


func testSyncRateLimiters(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		clock := timeutil.NewFakeClock(time.Now(), 0)
		now := clock.Now()
		ctx := context.Background()
		transact(ctx, store, func(t testing.TB, tx *repos.Store) {
			toCreate := 501 // Larger than default page size in order to test pagination
			services := make([]*types.ExternalService, 0, toCreate)
			for i := 0; i < toCreate; i++ {
				svc := &types.ExternalService{
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

			if err := tx.ExternalServiceStore.Upsert(ctx, services...); err != nil {
				t.Fatalf("failed to setup store: %v", err)
			}

			registry := ratelimit.NewRegistry()
			syncer := repos.NewRateLimitSyncer(registry, tx.ExternalServiceStore)
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

func testStoreEnqueueSyncJobs(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		clock := timeutil.NewFakeClock(time.Now(), 0)
		now := clock.Now()

		services := generateExternalServices(10, mkExternalServices(now)...)

		type testCase struct {
			name            string
			stored          types.ExternalServices
			queued          func(types.ExternalServices) []int64
			ignoreSiteAdmin bool
			err             error
		}

		var testCases []testCase

		testCases = append(testCases, testCase{
			name: "enqueue everything",
			stored: services.With(func(s *types.ExternalService) {
				s.NextSyncAt = now.Add(-10 * time.Second)
			}),
			queued: func(svcs types.ExternalServices) []int64 { return svcs.IDs() },
		})

		testCases = append(testCases, testCase{
			name: "nothing to enqueue",
			stored: services.With(func(s *types.ExternalService) {
				s.NextSyncAt = now.Add(10 * time.Second)
			}),
			queued: func(svcs types.ExternalServices) []int64 { return []int64{} },
		})

		testCases = append(testCases, testCase{
			name: "ignore siteadmin repos",
			stored: services.With(func(s *types.ExternalService) {
				s.NextSyncAt = now.Add(10 * time.Second)
			}),
			ignoreSiteAdmin: true,
			queued:          func(svcs types.ExternalServices) []int64 { return []int64{} },
		})

		{
			i := 0
			testCases = append(testCases, testCase{
				name: "some to enqueue",
				stored: services.With(func(s *types.ExternalService) {
					if i%2 == 0 {
						s.NextSyncAt = now.Add(10 * time.Second)
					} else {
						s.NextSyncAt = now.Add(-10 * time.Second)
					}
					i++
				}),
				queued: func(svcs types.ExternalServices) []int64 {
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

		for _, tc := range testCases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Cleanup(func() {
					if err := store.Exec(ctx, sqlf.Sprintf("DELETE FROM external_service_sync_jobs;DELETE FROM external_services")); err != nil {
						t.Fatal(err)
					}
				})
				stored := tc.stored.Clone()

				if err := store.ExternalServiceStore.Upsert(ctx, stored...); err != nil {
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

func testStoreEnqueueSingleSyncJob(store *repos.Store) func(*testing.T) {
	return func(t *testing.T) {
		clock := timeutil.NewFakeClock(time.Now(), 0)
		now := clock.Now()

		ctx := context.Background()
		t.Cleanup(func() {
			if err := store.Exec(ctx, sqlf.Sprintf("DELETE FROM external_service_sync_jobs;DELETE FROM external_services")); err != nil {
				t.Fatal(err)
			}
		})
		service := types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "Github - Test",
			Config:      `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "abc"}`,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Create a new external service
		confGet := func() *conf.Unified {
			return &conf.Unified{}
		}
		err := database.ExternalServicesWith(store).Create(ctx, confGet, &service)
		if err != nil {
			t.Fatal(err)
		}

		assertCount := func(t *testing.T, want int) {
			t.Helper()
			var count int
			if err := store.QueryRow(ctx, sqlf.Sprintf("SELECT COUNT(*) FROM external_service_sync_jobs")).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != want {
				t.Fatalf("Expected %d rows, got %d", want, count)
			}
		}
		assertCount(t, 0)

		err = store.EnqueueSingleSyncJob(ctx, service.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertCount(t, 1)

		// Doing it again should not fail or add a new row
		err = store.EnqueueSingleSyncJob(ctx, service.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertCount(t, 1)

		// If we change status to processing it should not add a new row
		if err := store.Exec(ctx, sqlf.Sprintf("UPDATE external_service_sync_jobs SET state='processing'")); err != nil {
			t.Fatal(err)
		}
		err = store.EnqueueSingleSyncJob(ctx, service.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertCount(t, 1)

		// If we change status to completed we should be able to enqueue another one
		if err = store.Exec(ctx, sqlf.Sprintf("UPDATE external_service_sync_jobs SET state='completed'")); err != nil {
			t.Fatal(err)
		}
		err = store.EnqueueSingleSyncJob(ctx, service.ID)
		if err != nil {
			t.Fatal(err)
		}
		assertCount(t, 2)
	}
}

func mkRepos(n int, base ...*types.Repo) types.Repos {
	if len(base) == 0 {
		return nil
	}

	rs := make(types.Repos, 0, n)
	for i := 0; i < n; i++ {
		id := strconv.Itoa(i)
		r := base[i%len(base)].Clone()
		r.Name += api.RepoName(id)
		r.ExternalRepo.ID += id
		rs = append(rs, r)
	}
	return rs
}

func generateExternalServices(n int, base ...*types.ExternalService) types.ExternalServices {
	if len(base) == 0 {
		return nil
	}
	es := make(types.ExternalServices, 0, n)
	for i := 0; i < n; i++ {
		id := strconv.Itoa(i)
		r := base[i%len(base)].Clone()
		r.DisplayName += id
		es = append(es, r)
	}
	return es
}

func transact(ctx context.Context, s *repos.Store, test func(testing.TB, *repos.Store)) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		var err error
		txStore := s

		if !s.InTransaction() {
			txStore, err = s.Transact(ctx)
			if err != nil {
				t.Fatalf("failed to start transaction: %v", err)
			}
			defer txStore.Done(errRollback)
		}

		test(t, txStore)
	}
}

func createExternalServices(t *testing.T, store *repos.Store, opts ...func(*types.ExternalService)) map[string]*types.ExternalService {
	clock := timeutil.NewFakeClock(time.Now(), 0)
	now := clock.Now()

	svcs := mkExternalServices(now)
	for _, svc := range svcs {
		for _, opt := range opts {
			opt(svc)
		}
	}

	// create a few external services
	if err := store.ExternalServiceStore.Upsert(context.Background(), svcs...); err != nil {
		t.Fatalf("failed to insert external services: %v", err)
	}

	services, err := store.ExternalServiceStore.List(context.Background(), database.ExternalServicesListOptions{})
	if err != nil {
		t.Fatal("failed to list external services")
	}

	servicesPerKind := make(map[string]*types.ExternalService)
	for _, svc := range services {
		servicesPerKind[svc.Kind] = svc
	}

	return servicesPerKind
}

func mkExternalServices(now time.Time) types.ExternalServices {
	githubSvc := types.ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "Github - Test",
		Config:      `{"url": "https://github.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	gitlabSvc := types.ExternalService{
		Kind:        extsvc.KindGitLab,
		DisplayName: "GitLab - Test",
		Config:      `{"url": "https://gitlab.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	bitbucketServerSvc := types.ExternalService{
		Kind:        extsvc.KindBitbucketServer,
		DisplayName: "Bitbucket Server - Test",
		Config:      `{"url": "https://bitbucket.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	bitbucketCloudSvc := types.ExternalService{
		Kind:        extsvc.KindBitbucketCloud,
		DisplayName: "Bitbucket Cloud - Test",
		Config:      `{"url": "https://bitbucket.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	awsSvc := types.ExternalService{
		Kind:        extsvc.KindAWSCodeCommit,
		DisplayName: "AWS Code - Test",
		Config:      `{"url": "https://aws.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	otherSvc := types.ExternalService{
		Kind:        extsvc.KindOther,
		DisplayName: "Other - Test",
		Config:      `{"url": "https://other.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	gitoliteSvc := types.ExternalService{
		Kind:        extsvc.KindGitolite,
		DisplayName: "Gitolite - Test",
		Config:      `{"url": "https://gitolite.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return []*types.ExternalService{
		&githubSvc,
		&gitlabSvc,
		&bitbucketServerSvc,
		&bitbucketCloudSvc,
		&awsSvc,
		&otherSvc,
		&gitoliteSvc,
	}
}
