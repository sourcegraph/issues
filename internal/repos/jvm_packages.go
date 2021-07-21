package repos

import (
	"context"
	"fmt"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/jvmpackages"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/jvmpackages/coursier"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

// A JVMPackagesSource creates git repositories from `*-sources.jar` files of
// published Maven dependencies from the JVM ecosystem.
type JVMPackagesSource struct {
	svc     *types.ExternalService
	config  *schema.JVMPackagesConnection
	dbStore JVMPackagesRepoStore
}

type JVMPackagesRepoStore interface {
	GetJVMDependencyRepos(ctx context.Context) ([]JVMDependencyRepo, error)
}

type JVMDependencyRepo struct {
	Identifier, Version string
}

// NewJVMPackagesSource returns a new MavenSource from the given external
// service.
func NewJVMPackagesSource(svc *types.ExternalService, dbStore JVMPackagesRepoStore) (*JVMPackagesSource, error) {
	var c schema.JVMPackagesConnection
	if err := jsonc.Unmarshal(svc.Config, &c); err != nil {
		return nil, fmt.Errorf("external service id=%d config error: %s", svc.ID, err)
	}
	return newJVMPackagesSource(svc, &c, dbStore)
}

func newJVMPackagesSource(svc *types.ExternalService, c *schema.JVMPackagesConnection, dbStore JVMPackagesRepoStore) (*JVMPackagesSource, error) {
	return &JVMPackagesSource{
		svc:     svc,
		config:  c,
		dbStore: dbStore,
	}, nil
}

// ListRepos returns all Maven artifacts accessible to all connections
// configured in Sourcegraph via the external services configuration.
func (s *JVMPackagesSource) ListRepos(ctx context.Context, results chan SourceResult) {
	s.listDependentRepos(ctx, results)
}

func (s *JVMPackagesSource) listDependentRepos(ctx context.Context, results chan SourceResult) {
	modules, err := MavenModules(*s.config)
	if err != nil {
		results <- SourceResult{Err: err}
		return
	}
	for _, module := range modules {
		repo := s.makeRepo(module)
		results <- SourceResult{
			Source: s,
			Repo:   repo,
		}
	}
	if err != nil {
		results <- SourceResult{Err: err}
		return
	}

	dbDeps, err := s.dbStore.GetJVMDependencyRepos(ctx)
	if err != nil {
		results <- SourceResult{Err: err}
		return
	}

	for _, dep := range dbDeps {
		parsed, err := reposource.ParseMavenDependency(dep.Identifier)
		if err != nil {
			continue
		}
		repo := s.makeRepo(parsed.MavenModule)
		results <- SourceResult{
			Source: s,
			Repo:   repo,
		}
	}
}

func (s *JVMPackagesSource) GetRepo(ctx context.Context, artifactPath string) (*types.Repo, error) {
	module, err := reposource.ParseMavenModule(artifactPath)
	if err != nil {
		return nil, err
	}

	dependencies, err := MavenDependencies(*s.config)
	if err != nil {
		return nil, err
	}

	dbDeps, err := s.dbStore.GetJVMDependencyRepos(ctx)
	if err != nil {
		return nil, err
	}

	for _, dep := range dbDeps {
		parsed, err := reposource.ParseMavenDependency(dep.Identifier)
		if err != nil {
			continue
		}
		dependencies = append(dependencies, parsed)
	}

	for _, dep := range dependencies {
		if dep.MavenModule == module {
			exists, err := coursier.Exists(ctx, s.config, dep)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, &mavenArtifactNotFound{
					dependency: dep,
				}
			}
		}
	}

	return s.makeRepo(module), nil
}

type mavenArtifactNotFound struct {
	dependency reposource.MavenDependency
}

func (mavenArtifactNotFound) NotFound() bool {
	return true
}

func (e *mavenArtifactNotFound) Error() string {
	return fmt.Sprintf("not found: maven dependency '%v'", e.dependency)
}

func (s *JVMPackagesSource) makeRepo(module reposource.MavenModule) *types.Repo {
	urn := s.svc.URN()
	cloneURL := module.CloneURL()
	return &types.Repo{
		Name: module.RepoName(),
		URI:  string(module.RepoName()),
		ExternalRepo: api.ExternalRepoSpec{
			ID:          string(module.RepoName()),
			ServiceID:   extsvc.TypeJVMPackages,
			ServiceType: extsvc.TypeJVMPackages,
		},
		Private: false,
		Sources: map[string]*types.SourceInfo{
			urn: {
				ID:       urn,
				CloneURL: cloneURL,
			},
		},
		Metadata: &jvmpackages.Metadata{
			Module: module,
		},
	}
}

// ExternalServices returns a singleton slice containing the external service.
func (s *JVMPackagesSource) ExternalServices() types.ExternalServices {
	return types.ExternalServices{s.svc}
}

func MavenDependencies(connection schema.JVMPackagesConnection) (dependencies []reposource.MavenDependency, err error) {
	for _, dep := range connection.Maven.Dependencies {
		dependency, err := reposource.ParseMavenDependency(dep)
		if err != nil {
			return nil, err
		}
		dependencies = append(dependencies, dependency)
	}
	return dependencies, nil
}

func MavenModules(connection schema.JVMPackagesConnection) ([]reposource.MavenModule, error) {
	isAdded := make(map[reposource.MavenModule]bool)
	modules := []reposource.MavenModule{}
	dependencies, err := MavenDependencies(connection)
	if err != nil {
		return nil, err
	}
	for _, dep := range dependencies {
		module := dep.MavenModule
		if _, added := isAdded[module]; !added {
			modules = append(modules, module)
		}
		isAdded[module] = true
	}
	return modules, nil
}
