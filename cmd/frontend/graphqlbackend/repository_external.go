package graphqlbackend

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
)

func (r *RepositoryResolver) ExternalRepository() *externalRepositoryResolver {
	return &externalRepositoryResolver{repository: r}
}

type externalRepositoryResolver struct {
	repository *RepositoryResolver
}

func (r *externalRepositoryResolver) ID() string { return r.repository.repo.ExternalRepo.ID }
func (r *externalRepositoryResolver) ServiceType() string {
	return r.repository.repo.ExternalRepo.ServiceType
}

func (r *externalRepositoryResolver) ServiceID() string {
	return r.repository.repo.ExternalRepo.ServiceID
}

func (r *RepositoryResolver) ExternalServices(ctx context.Context, args *struct {
	graphqlutil.ConnectionArgs
}) (*computedExternalServiceConnectionResolver, error) {
	// 🚨 SECURITY: Only site admins may read external services (they have secrets).
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	svcs, err := repoupdater.DefaultClient.RepoExternalServices(ctx, r.repo.ID)
	if err != nil {
		return nil, err
	}

	return &computedExternalServiceConnectionResolver{
		args:             args.ConnectionArgs,
		externalServices: newExternalServices(svcs...),
	}, nil
}

func newExternalServices(es ...api.ExternalService) []*types.ExternalService {
	svcs := make([]*types.ExternalService, 0, len(es))

	for _, e := range es {
		svc := &types.ExternalService{
			ID:              e.ID,
			Kind:            e.Kind,
			DisplayName:     e.DisplayName,
			Config:          e.Config,
			CreatedAt:       e.CreatedAt,
			UpdatedAt:       e.UpdatedAt,
			DeletedAt:       e.DeletedAt,
			LastSyncAt:      e.LastSyncAt,
			NextSyncAt:      e.NextSyncAt,
			NamespaceUserID: e.NamespaceUserID,
		}

		svcs = append(svcs, svc)
	}

	return svcs
}
