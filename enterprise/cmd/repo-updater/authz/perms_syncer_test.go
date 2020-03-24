package authz

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/authz"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	edb "github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/db"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
)

func TestPermsSyncer_ScheduleUsers(t *testing.T) {
	s := NewPermsSyncer(nil, nil, nil)
	s.ScheduleUsers(context.Background(), PriorityHigh, 1)

	expHeap := []*syncRequest{
		{requestMeta: &requestMeta{
			Priority: PriorityHigh,
			Type:     requestTypeUser,
			ID:       1,
		}, acquired: false, index: 0},
	}
	if diff := cmp.Diff(expHeap, s.queue.heap, cmpOpts); diff != "" {
		t.Fatalf("heap: %v", diff)
	}
}

func TestPermsSyncer_ScheduleRepos(t *testing.T) {
	s := NewPermsSyncer(nil, nil, nil)
	s.ScheduleRepos(context.Background(), PriorityHigh, 1)

	expHeap := []*syncRequest{
		{requestMeta: &requestMeta{
			Priority: PriorityHigh,
			Type:     requestTypeRepo,
			ID:       1,
		}, acquired: false, index: 0},
	}
	if diff := cmp.Diff(expHeap, s.queue.heap, cmpOpts); diff != "" {
		t.Fatalf("heap: %v", diff)
	}
}

type mockProvider struct {
	serviceType string
	serviceID   string

	fetchUserPerms func(context.Context, *extsvc.ExternalAccount) ([]extsvc.ExternalRepoID, error)
	fetchRepoPerms func(ctx context.Context, repo *api.ExternalRepoSpec) ([]extsvc.ExternalAccountID, error)
}

func (*mockProvider) RepoPerms(context.Context, *extsvc.ExternalAccount, []*types.Repo) ([]authz.RepoPerms, error) {
	return nil, nil
}

func (*mockProvider) FetchAccount(context.Context, *types.User, []*extsvc.ExternalAccount) (*extsvc.ExternalAccount, error) {
	return nil, nil
}

func (p *mockProvider) ServiceType() string {
	return p.serviceType
}

func (p *mockProvider) ServiceID() string {
	return p.serviceID
}

func (*mockProvider) Validate() []string {
	return nil
}

func (p *mockProvider) FetchUserPerms(ctx context.Context, acct *extsvc.ExternalAccount) ([]extsvc.ExternalRepoID, error) {
	return p.fetchUserPerms(ctx, acct)
}

func (p *mockProvider) FetchRepoPerms(ctx context.Context, repo *api.ExternalRepoSpec) ([]extsvc.ExternalAccountID, error) {
	return p.fetchRepoPerms(ctx, repo)
}

type mockReposStore struct {
	listRepos func(context.Context, repos.StoreListReposArgs) ([]*repos.Repo, error)
}

func (s *mockReposStore) ListExternalServices(context.Context, repos.StoreListExternalServicesArgs) ([]*repos.ExternalService, error) {
	return nil, nil
}

func (s *mockReposStore) UpsertExternalServices(context.Context, ...*repos.ExternalService) error {
	return nil
}

func (s *mockReposStore) ListRepos(ctx context.Context, args repos.StoreListReposArgs) ([]*repos.Repo, error) {
	return s.listRepos(ctx, args)
}

func (s *mockReposStore) UpsertRepos(context.Context, ...*repos.Repo) error {
	return nil
}

func (s *mockReposStore) ListAllRepoNames(context.Context) ([]api.RepoName, error) {
	return nil, nil
}

type mockPermsStore struct {
	listExternalAccounts func(context.Context, int32) ([]*extsvc.ExternalAccount, error)
}

func (s *mockPermsStore) ListExternalAccounts(ctx context.Context, userID int32) ([]*extsvc.ExternalAccount, error) {
	return s.listExternalAccounts(ctx, userID)
}

func TestPermsSyncer_syncUserPerms(t *testing.T) {
	p := &mockProvider{
		serviceType: gitlab.ServiceType,
		serviceID:   "https://gitlab.com/",
	}
	authz.SetProviders(false, []authz.Provider{p})
	defer authz.SetProviders(true, nil)

	extAccount := extsvc.ExternalAccount{
		ExternalAccountSpec: extsvc.ExternalAccountSpec{
			ServiceType: p.ServiceType(),
			ServiceID:   p.ServiceID(),
		},
	}
	p.fetchUserPerms = func(context.Context, *extsvc.ExternalAccount) ([]extsvc.ExternalRepoID, error) {
		return []extsvc.ExternalRepoID{"1"}, nil
	}

	edb.Mocks.Perms.ListExternalAccounts = func(context.Context, int32) ([]*extsvc.ExternalAccount, error) {
		return []*extsvc.ExternalAccount{&extAccount}, nil
	}
	edb.Mocks.Perms.SetUserPermissions = func(_ context.Context, p *authz.UserPermissions) error {
		if p.UserID != 1 {
			return fmt.Errorf("UserID: want 1 but got %d", p.UserID)
		}

		expIDs := []uint32{1}
		if diff := cmp.Diff(expIDs, p.IDs.ToArray()); diff != "" {
			return fmt.Errorf("IDs: %v", diff)
		}
		return nil
	}
	defer func() {
		edb.Mocks.Perms = edb.MockPerms{}
	}()

	reposStore := &mockReposStore{
		listRepos: func(context.Context, repos.StoreListReposArgs) ([]*repos.Repo, error) {
			return []*repos.Repo{{ID: 1}}, nil
		},
	}
	clock := func() time.Time {
		return time.Now().UTC().Truncate(time.Microsecond)
	}
	permsStore := edb.NewPermsStore(nil, clock)
	s := NewPermsSyncer(reposStore, permsStore, clock)
	s.metrics.syncDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{}, []string{"type", "success"})
	s.metrics.syncErrors = prometheus.NewCounterVec(prometheus.CounterOpts{}, []string{"type"})

	err := s.syncUserPerms(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPermsSyncer_syncRepoPerms(t *testing.T) {

}

func TestPermsSyncer_syncPerms(t *testing.T) {

}

func TestPermsSyncer_runSync(t *testing.T) {

}

func TestPermsSyncer_schedule(t *testing.T) {

}
