package repos

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/maven/coursier"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

// A MavenSource yields depots from a single Maven connection configured
// in Sourcegraph via the external services configuration.
type MavenSource struct {
	svc    *types.ExternalService
	config *schema.MavenConnection
}

// NewMavenSource returns a new MavenSource from the given external
// service.
func NewMavenSource(svc *types.ExternalService) (*MavenSource, error) {
	var c schema.MavenConnection
	if err := jsonc.Unmarshal(svc.Config, &c); err != nil {
		return nil, fmt.Errorf("external service id=%d config error: %s", svc.ID, err)
	}
	return newMavenSource(svc, &c)
}

func newMavenSource(svc *types.ExternalService, c *schema.MavenConnection) (*MavenSource, error) {
	return &MavenSource{
		svc:    svc,
		config: c,
	}, nil
}

// ListRepos returns all Maven artifacts accessible to all connections
// configured in Sourcegraph via the external services configuration.
func (s MavenSource) ListRepos(ctx context.Context, results chan SourceResult) {
	s.listDependentRepos(ctx, results)
}

func (s MavenSource) listDependentRepos(ctx context.Context, results chan SourceResult) {
	listed := make(map[string]struct{})
	for _, artifact := range s.config.Artifacts {
		split := strings.Split(artifact, ":")
		groupID := split[0]
		artifactID := split[1]
		versions, err := coursier.ListVersions(ctx, s.config, groupID, artifactID)
		if err != nil {
			results <- SourceResult{Err: err}
			continue
		}
		if len(versions) == 0 {
			results <- SourceResult{
				Err: &mavenArtifactNotFound{groupID: groupID, artifactID: artifactID},
			}
			continue
		}

		listed[artifact] = struct{}{}
		results <- SourceResult{
			Source: s,
			Repo:   s.makeRepo(groupID, artifactID, versions[len(versions)-1]),
		}
	}

	for _, groupID := range s.config.Groups {
		artifacts, err := coursier.ListArtifactIDs(ctx, s.config, groupID)
		if err != nil {
			results <- SourceResult{Err: err}
			continue
		}

		for _, artifactID := range artifacts {
			versions, err := coursier.ListVersions(ctx, s.config, groupID, artifactID)
			if err != nil {
				results <- SourceResult{Err: err}
			}
			if len(versions) == 0 {
				results <- SourceResult{
					Err: &mavenArtifactNotFound{
						groupID:    groupID,
						artifactID: artifactID,
					},
				}
			}
			results <- SourceResult{
				Source: s,
				Repo:   s.makeRepo(groupID, artifactID, versions[len(versions)-1]),
			}
		}
	}
}

func (s MavenSource) GetRepo(ctx context.Context, artifactPath string) (*types.Repo, error) {
	groupID, artifactID, version := reposource.DecomposeMavenPath(artifactPath)

	exists, err := coursier.Exists(ctx, s.config, groupID, artifactID, version)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &mavenArtifactNotFound{
			groupID:    groupID,
			artifactID: artifactID,
		}
	}

	return s.makeRepo(groupID, artifactID, version), nil
}

type mavenArtifactNotFound struct {
	groupID    string
	artifactID string
}

func (mavenArtifactNotFound) NotFound() bool {
	return true
}

func (e *mavenArtifactNotFound) Error() string {
	return fmt.Sprintf("maven artifact %v:%v not found", e.groupID, e.artifactID)
}

func (s MavenSource) makeRepo(groupID, artifactID, version string) *types.Repo {
	fullArtifactID := strings.Join([]string{groupID, artifactID, version}, ":")
	repoName := "maven/" + fullArtifactID
	urn := s.svc.URN()
	cloneURL := url.URL{
		Host: "maven",
		Path: repoName,
	}
	return &types.Repo{
		Name: api.RepoName(repoName),
		URI:  repoName,
		ExternalRepo: api.ExternalRepoSpec{
			ID:          fullArtifactID,
			ServiceType: extsvc.TypeMaven,
		},
		Private: false,
		Sources: map[string]*types.SourceInfo{
			urn: {
				ID:       urn,
				CloneURL: cloneURL.String(),
			},
		},
		Metadata: nil,
	}
}

// ExternalServices returns a singleton slice containing the external service.
func (s MavenSource) ExternalServices() types.ExternalServices {
	return types.ExternalServices{s.svc}
}
