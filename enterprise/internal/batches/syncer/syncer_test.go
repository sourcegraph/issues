package syncer

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestSyncerRun(t *testing.T) {
	t.Parallel()

	t.Run("Sync due", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		now := time.Now()
		syncStore := MockSyncStore{
			listChangesetSyncData: func(ctx context.Context, opts store.ListChangesetSyncDataOpts) ([]*btypes.ChangesetSyncData, error) {
				return []*btypes.ChangesetSyncData{
					{
						ChangesetID:       1,
						UpdatedAt:         now.Add(-2 * maxSyncDelay),
						LatestEvent:       now.Add(-2 * maxSyncDelay),
						ExternalUpdatedAt: now.Add(-2 * maxSyncDelay),
					},
				}, nil
			},
		}
		syncFunc := func(ctx context.Context, ids int64) error {
			cancel()
			return nil
		}
		syncer := &changesetSyncer{
			syncStore:        syncStore,
			scheduleInterval: 10 * time.Minute,
			syncFunc:         syncFunc,
		}
		go syncer.Run(ctx)
		select {
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Sync should have been triggered")
		}
	})

	t.Run("Sync due but reenqueued for reconciler", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		now := time.Now()
		updateCalled := false
		syncStore := MockSyncStore{
			getChangeset: func(context.Context, store.GetChangesetOpts) (*btypes.Changeset, error) {
				// Return ErrNoResults, which is the result you get when the changeset preconditions aren't met anymore.
				// The sync data checks for the reconciler state and if it changed since the sync data was loaded,
				// we don't get back the changeset here and skip it.
				//
				// If we don't return ErrNoResults, the rest of the test will fail, because not all
				// methods of sync store are mocked.
				return nil, store.ErrNoResults
			},
			updateChangesetCodeHostState: func(context.Context, *btypes.Changeset) error {
				updateCalled = true
				return nil
			},
			listChangesetSyncData: func(ctx context.Context, opts store.ListChangesetSyncDataOpts) ([]*btypes.ChangesetSyncData, error) {
				return []*btypes.ChangesetSyncData{
					{
						ChangesetID:       1,
						UpdatedAt:         now.Add(-2 * maxSyncDelay),
						LatestEvent:       now.Add(-2 * maxSyncDelay),
						ExternalUpdatedAt: now.Add(-2 * maxSyncDelay),
					},
				}, nil
			},
		}
		syncer := &changesetSyncer{
			syncStore:        syncStore,
			scheduleInterval: 10 * time.Minute,
		}
		syncer.Run(ctx)
		if updateCalled {
			t.Fatal("Called UpdateChangeset, but shouldn't have")
		}
	})

	t.Run("Sync not due", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		now := time.Now()
		syncStore := MockSyncStore{
			listChangesetSyncData: func(ctx context.Context, opts store.ListChangesetSyncDataOpts) ([]*btypes.ChangesetSyncData, error) {
				return []*btypes.ChangesetSyncData{
					{
						ChangesetID:       1,
						UpdatedAt:         now,
						LatestEvent:       now,
						ExternalUpdatedAt: now,
					},
				}, nil
			},
		}
		var syncCalled bool
		syncFunc := func(ctx context.Context, ids int64) error {
			syncCalled = true
			return nil
		}
		syncer := &changesetSyncer{
			syncStore:        syncStore,
			scheduleInterval: 10 * time.Minute,
			syncFunc:         syncFunc,
		}
		syncer.Run(ctx)
		if syncCalled {
			t.Fatal("Sync should not have been triggered")
		}
	})

	t.Run("Priority added", func(t *testing.T) {
		// Empty schedule but then we add an item
		ctx, cancel := context.WithCancel(context.Background())
		syncStore := MockSyncStore{
			listChangesetSyncData: func(ctx context.Context, opts store.ListChangesetSyncDataOpts) ([]*btypes.ChangesetSyncData, error) {
				return []*btypes.ChangesetSyncData{}, nil
			},
		}
		syncFunc := func(ctx context.Context, ids int64) error {
			cancel()
			return nil
		}
		syncer := &changesetSyncer{
			syncStore:        syncStore,
			scheduleInterval: 10 * time.Minute,
			syncFunc:         syncFunc,
			priorityNotify:   make(chan []int64, 1),
		}
		syncer.priorityNotify <- []int64{1}
		go syncer.Run(ctx)
		select {
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Sync not called")
		}
	})
}

func TestSyncRegistry(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	now := time.Now()

	externalServiceID := "https://example.com/"

	codeHosts := []*btypes.CodeHost{{ExternalServiceID: externalServiceID, ExternalServiceType: extsvc.TypeGitHub}}

	syncStore := MockSyncStore{
		listChangesetSyncData: func(ctx context.Context, opts store.ListChangesetSyncDataOpts) (data []*btypes.ChangesetSyncData, err error) {
			return []*btypes.ChangesetSyncData{
				{
					ChangesetID:           1,
					UpdatedAt:             now,
					RepoExternalServiceID: externalServiceID,
				},
			}, nil
		},
		listCodeHosts: func(c context.Context, lcho store.ListCodeHostsOpts) ([]*btypes.CodeHost, error) {
			return codeHosts, nil
		},
	}

	r := NewSyncRegistry(ctx, syncStore, nil)

	assertSyncerCount := func(want int) {
		r.mu.Lock()
		if len(r.syncers) != want {
			t.Fatalf("Expected %d syncer, got %d", want, len(r.syncers))
		}
		r.mu.Unlock()
	}

	assertSyncerCount(1)

	// Adding it again should have no effect
	r.Add(&btypes.CodeHost{ExternalServiceID: "https://example.com/", ExternalServiceType: extsvc.TypeGitHub})
	assertSyncerCount(1)

	// Simulate a service being removed
	oldCodeHosts := codeHosts
	codeHosts = []*btypes.CodeHost{}
	r.HandleExternalServiceSync(api.ExternalService{
		ID:        1,
		Kind:      extsvc.KindGitHub,
		Config:    `{"url": "https://example.com/"}`,
		DeletedAt: now,
	})
	assertSyncerCount(0)
	codeHosts = oldCodeHosts

	// And added again
	r.HandleExternalServiceSync(api.ExternalService{
		ID:   1,
		Kind: extsvc.KindGitHub,
	})
	assertSyncerCount(1)

	syncChan := make(chan int64, 1)

	// In order to test that priority items are delivered we'll inject our own syncer
	// with a custom sync func
	syncer := &changesetSyncer{
		syncStore:   syncStore,
		codeHostURL: "https://example.com/",
		syncFunc: func(ctx context.Context, id int64) error {
			syncChan <- id
			return nil
		},
		priorityNotify: make(chan []int64, 1),
	}
	go syncer.Run(ctx)

	// Set the syncer
	r.mu.Lock()
	r.syncers["https://example.com/"] = syncer
	r.mu.Unlock()

	// Send priority items
	err := r.EnqueueChangesetSyncs(ctx, []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case id := <-syncChan:
		if id != 1 {
			t.Fatalf("Expected 1, got %d", id)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for sync")
	}
}

func TestLoadChangesetSource(t *testing.T) {
	ctx := context.Background()
	cf := httpcli.NewFactory(
		func(cli httpcli.Doer) httpcli.Doer {
			return httpcli.DoerFunc(func(req *http.Request) (*http.Response, error) {
				// Don't actually execute the request, just dump the authorization header
				// in the error, so we can assert on it further down.
				return nil, errors.New(req.Header.Get("Authorization"))
			})
		},
		httpcli.NewTimeoutOpt(1*time.Second),
	)

	externalService := types.ExternalService{
		ID:          1,
		Kind:        extsvc.KindGitHub,
		DisplayName: "GitHub.com",
		Config:      `{"url": "https://github.com", "token": "123", "authorization": {}}`,
	}
	repo := &types.Repo{
		Name:    api.RepoName("test-repo"),
		URI:     "test-repo",
		Private: true,
		ExternalRepo: api.ExternalRepoSpec{
			ID:          "external-id-123",
			ServiceType: extsvc.TypeGitHub,
			ServiceID:   "https://github.com/",
		},
		Sources: map[string]*types.SourceInfo{
			externalService.URN(): {
				ID:       externalService.URN(),
				CloneURL: "https://123@github.com/sourcegraph/sourcegraph",
			},
		},
	}

	// Store mocks.
	database.Mocks.ExternalServices.List = func(opt database.ExternalServicesListOptions) ([]*types.ExternalService, error) {
		return []*types.ExternalService{&externalService}, nil
	}
	t.Cleanup(func() {
		database.Mocks.ExternalServices.List = nil
	})
	hasCredential := false
	syncStore := &MockSyncStore{
		getSiteCredential: func(ctx context.Context, opts store.GetSiteCredentialOpts) (*btypes.SiteCredential, error) {
			if hasCredential {
				cred := &btypes.SiteCredential{}
				cred.SetAuthenticator(ctx, &auth.OAuthBearerToken{Token: "456"})
				return cred, nil
			}
			return nil, store.ErrNoResults
		},
	}

	// If no site-credential exists, the token from the external service should be used.
	src, err := loadChangesetSource(ctx, cf, syncStore, repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := src.ValidateAuthenticator(ctx); err == nil {
		t.Fatal("unexpected nil error")
	} else if have, want := err.Error(), "Bearer 123"; have != want {
		t.Fatalf("invalid token used, want=%q have=%q", want, have)
	}

	// If one exists, prefer that one over the external service config ones.
	hasCredential = true
	src, err = loadChangesetSource(ctx, cf, syncStore, repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := src.ValidateAuthenticator(ctx); err == nil {
		t.Fatal("unexpected nil error")
	} else if have, want := err.Error(), "Bearer 456"; have != want {
		t.Fatalf("invalid token used, want=%q have=%q", want, have)
	}
}

type MockSyncStore struct {
	listCodeHosts                func(context.Context, store.ListCodeHostsOpts) ([]*btypes.CodeHost, error)
	listChangesetSyncData        func(context.Context, store.ListChangesetSyncDataOpts) ([]*btypes.ChangesetSyncData, error)
	getChangeset                 func(context.Context, store.GetChangesetOpts) (*btypes.Changeset, error)
	updateChangesetCodeHostState func(context.Context, *btypes.Changeset) error
	upsertChangesetEvents        func(context.Context, ...*btypes.ChangesetEvent) error
	getSiteCredential            func(ctx context.Context, opts store.GetSiteCredentialOpts) (*btypes.SiteCredential, error)
	getExternalServiceIDs        func(ctx context.Context, opts store.GetExternalServiceIDsOpts) ([]int64, error)
	transact                     func(context.Context) (*store.Store, error)
}

func (m MockSyncStore) ListChangesetSyncData(ctx context.Context, opts store.ListChangesetSyncDataOpts) ([]*btypes.ChangesetSyncData, error) {
	return m.listChangesetSyncData(ctx, opts)
}

func (m MockSyncStore) GetChangeset(ctx context.Context, opts store.GetChangesetOpts) (*btypes.Changeset, error) {
	return m.getChangeset(ctx, opts)
}

func (m MockSyncStore) UpdateChangesetCodeHostState(ctx context.Context, c *btypes.Changeset) error {
	return m.updateChangesetCodeHostState(ctx, c)
}

func (m MockSyncStore) UpsertChangesetEvents(ctx context.Context, cs ...*btypes.ChangesetEvent) error {
	return m.upsertChangesetEvents(ctx, cs...)
}

func (m MockSyncStore) GetSiteCredential(ctx context.Context, opts store.GetSiteCredentialOpts) (*btypes.SiteCredential, error) {
	return m.getSiteCredential(ctx, opts)
}

func (m MockSyncStore) GetExternalServiceIDs(ctx context.Context, opts store.GetExternalServiceIDsOpts) ([]int64, error) {
	return m.getExternalServiceIDs(ctx, opts)
}

func (m MockSyncStore) Transact(ctx context.Context) (*store.Store, error) {
	return m.transact(ctx)
}

func (m MockSyncStore) Repos() *database.RepoStore {
	// Return a RepoStore with a nil DB, so tests will fail when a mock is missing.
	return database.Repos(&dbtesting.MockDB{})
}

func (m MockSyncStore) ExternalServices() *database.ExternalServiceStore {
	// Return a ExternalServiceStore with a nil DB, so tests will fail when a mock is missing.
	return database.ExternalServices(&dbtesting.MockDB{})
}

func (m MockSyncStore) UserCredentials() *database.UserCredentialsStore {
	// Return a UserCredentialsStore with a nil DB, so tests will fail when a mock is missing.
	return database.UserCredentials(&dbtesting.MockDB{}, nil)
}

func (m MockSyncStore) DB() dbutil.DB {
	// Return a nil DB, so tests will fail when a mock is missing.
	return nil
}

func (m MockSyncStore) Clock() func() time.Time {
	return timeutil.Now
}

func (m MockSyncStore) ListCodeHosts(ctx context.Context, opts store.ListCodeHostsOpts) ([]*btypes.CodeHost, error) {
	return m.listCodeHosts(ctx, opts)
}
