package dependencyrepos

import (
	"context"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

type DBStore interface {
	ReferencesForUpload(ctx context.Context, uploadID int) (dbstore.PackageReferenceScanner, error)
	InsertCloneableDependencyRepos(ctx context.Context, dependencies []dbstore.DependencyRepoInfo) error
	ListExternalServices(ctx context.Context, opt database.ExternalServicesListOptions) ([]*types.ExternalService, error)
}

type DBStoreShim struct {
	*dbstore.Store
	ExternalServiceStore *database.ExternalServiceStore
}

func (s *DBStoreShim) ListExternalServices(ctx context.Context, opt database.ExternalServicesListOptions) ([]*types.ExternalService, error) {
	return s.ExternalServiceStore.List(ctx, opt)
}

type RepoUpdaterClient interface {
	SyncExternalService(ctx context.Context, svc api.ExternalService) (*protocol.ExternalServiceSyncResult, error)
}
