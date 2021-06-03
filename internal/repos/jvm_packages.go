package repos

import (
	"context"
	"fmt"
	"net/url"

	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/jvmpackages"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/jvmpackages/coursier"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

// A JvmPackagesSource yields depots from a single Maven connection configured
// in Sourcegraph via the external services configuration.
type JvmPackagesSource struct {
	svc    *types.ExternalService
	config *schema.JvmPackagesConnection
}

// NewJvmPackagesSource returns a new MavenSource from the given external
// service.
func NewJvmPackagesSource(svc *types.ExternalService) (*JvmPackagesSource, error) {
	var c schema.JvmPackagesConnection
	if err := jsonc.Unmarshal(svc.Config, &c); err != nil {
		return nil, fmt.Errorf("external service id=%d config error: %s", svc.ID, err)
	}
	return newJvmPackagesSource(svc, &c)
}

func newJvmPackagesSource(svc *types.ExternalService, c *schema.JvmPackagesConnection) (*JvmPackagesSource, error) {
	return &JvmPackagesSource{
		svc:    svc,
		config: c,
	}, nil
}

// ListRepos returns all Maven artifacts accessible to all connections
// configured in Sourcegraph via the external services configuration.
func (s JvmPackagesSource) ListRepos(ctx context.Context, results chan SourceResult) {
	s.listDependentRepos(ctx, results)
}

func (s JvmPackagesSource) listDependentRepos(ctx context.Context, results chan SourceResult) {
	for _, dependency := range s.config.Artifacts {
		repo := s.makeRepo(dependency)
		log15.Info("listDependentRepos", "repo", repo, "metadata", repo.Metadata)
		results <- SourceResult{
			Source: s,
			Repo:   repo,
		}
	}
}

func (s JvmPackagesSource) GetRepo(ctx context.Context, artifactPath string) (*types.Repo, error) {
	dependency := reposource.DecomposeMavenPath(artifactPath)

	exists, err := coursier.Exists(ctx, s.config, dependency)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &mavenArtifactNotFound{
			dependency: dependency,
		}
	}

	return s.makeRepo(dependency), nil
}

type mavenArtifactNotFound struct {
	dependency string
}

func (mavenArtifactNotFound) NotFound() bool {
	return true
}

func (e *mavenArtifactNotFound) Error() string {
	return fmt.Sprintf("not found: maven dependency '%v'", e.dependency)
}

func MavenRepoName(dependency string) string {
	return "maven/" + dependency
}

func MavenCloneURL(dependency string) string {
	cloneURL := url.URL{
		Host: "maven",
		Path: MavenRepoName(dependency),
	}
	return cloneURL.String()
}

func (s JvmPackagesSource) makeRepo(dependency string) *types.Repo {
	repoName := MavenRepoName(dependency)
	urn := s.svc.URN()
	cloneURL := MavenCloneURL(dependency)
	log15.Info("maven", "cloneURL", cloneURL)
	return &types.Repo{
		Name: api.RepoName(repoName),
		URI:  repoName,
		ExternalRepo: api.ExternalRepoSpec{
			ID:          dependency,
			ServiceID:   extsvc.TypeJvmPackages,
			ServiceType: extsvc.TypeJvmPackages,
		},
		Private: false,
		Sources: map[string]*types.SourceInfo{
			urn: {
				ID:       urn,
				CloneURL: cloneURL,
			},
		},
		Metadata: &jvmpackages.MavenMetadata{
			Dependency: dependency,
		},
	}
}

// ExternalServices returns a singleton slice containing the external service.
func (s JvmPackagesSource) ExternalServices() types.ExternalServices {
	return types.ExternalServices{s.svc}
}
