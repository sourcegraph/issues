package repos

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/sourcegraph/sourcegraph/pkg/api"
	"github.com/sourcegraph/sourcegraph/pkg/extsvc/github"
	"github.com/sourcegraph/sourcegraph/pkg/jsonc"
	"github.com/sourcegraph/sourcegraph/schema"
)

// A Sourcer yields Sources whose Repos should be synced.
type Sourcer interface {
	ListSources(context.Context) ([]Source, error)
}

// ExternalServicesSourcer converts each code host connection configured via external services
// in the frontend API to a Source that yields Repos. Each invocation of ListSources
// may yield different Sources depending on what the user configured at a given point in time.
type ExternalServicesSourcer struct {
	api   InternalAPI
	kinds []string
}

// NewExternalServicesSourcer returns a new ExternalServicesSourcer with the given Frontend API.
func NewExternalServicesSourcer(api InternalAPI, kinds ...string) *ExternalServicesSourcer {
	return &ExternalServicesSourcer{api: api, kinds: kinds}
}

// ListSources lists all configured repository yielding Sources of the configured kinds (via the constructor),
// based on the code host connections configured via external services in the frontend API.
func (s ExternalServicesSourcer) ListSources(ctx context.Context) ([]Source, error) {
	svcs, err := s.api.ExternalServicesList(ctx, api.ExternalServicesListRequest{Kinds: s.kinds})
	if err != nil {
		return nil, err
	}

	srcs := make([]Source, 0, len(svcs)+1)
	errs := new(multierror.Error)
	for _, svc := range svcs {
		if src, err := NewSource(svc); err != nil {
			errs = multierror.Append(errs, err)
		} else {
			srcs = append(srcs, src)
		}
	}

	if !includesGitHubDotComSource(srcs) {
		// add a GitHub.com source by default, to support navigating to URL paths like
		// /github.com/foo/bar to auto-add that repository.
		src, err := NewGithubDotComSource()
		srcs, errs = append(srcs, src), multierror.Append(errs, err)
	}

	return srcs, errs.ErrorOrNil()
}

// NewSource returns a repository yielding Source from the given api.ExternalService configuration.
func NewSource(svc *api.ExternalService) (Source, error) {
	switch svc.Kind {
	case "GITHUB":
		return NewGithubSource(svc)
	default:
		panic(fmt.Sprintf("source not implemented for external service kind %q", svc.Kind))
	}
}

func includesGitHubDotComSource(srcs []Source) bool {
	for _, src := range srcs {
		if gs, ok := src.(*GithubSource); !ok {
			continue
		} else if u, err := url.Parse(gs.conn.config.Url); err != nil {
			continue
		} else if strings.HasSuffix(u.Hostname(), "github.com") {
			return true
		}
	}
	return false
}

// A Source yields repositories to be stored and analysed by Sourcegraph.
// Successive calls to its ListRepos method may yield different results.
type Source interface {
	ListRepos(context.Context) ([]*Repo, error)
}

// A GithubSource yields repositories from a single Github connection configured
// in Sourcegraph via the external services configuration.
type GithubSource struct {
	svc  *api.ExternalService
	conn *githubConnection
}

// NewGithubSource returns a new GithubSource from the given external service.
func NewGithubSource(svc *api.ExternalService) (*GithubSource, error) {
	var c schema.GitHubConnection
	if err := jsonc.Unmarshal(svc.Config, &c); err != nil {
		return nil, fmt.Errorf("external service id=%d config error: %s", svc.ID, err)
	}
	return newGithubSource(svc, &c)
}

// NewGithubDotComSource returns a GithubSource for github.com, meant to be added
// to the list of sources in Sourcer when one isn't already configured in order to
// support navigating to URL paths like /github.com/foo/bar to auto-add that repository.
func NewGithubDotComSource() (*GithubSource, error) {
	svc := api.ExternalService{Kind: "GITHUB"}
	return newGithubSource(&svc, &schema.GitHubConnection{
		RepositoryQuery:             []string{"none"}, // don't try to list all repositories during syncs
		Url:                         "https://github.com",
		InitialRepositoryEnablement: true,
	})
}

func newGithubSource(svc *api.ExternalService, c *schema.GitHubConnection) (*GithubSource, error) {
	conn, err := newGitHubConnection(c)
	if err != nil {
		return nil, err
	}
	return &GithubSource{svc: svc, conn: conn}, nil
}

// ListRepos returns all Github repositories accessible to all connections configured
// in Sourcegraph via the external services configuration.
func (s GithubSource) ListRepos(ctx context.Context) ([]*Repo, error) {
	var repos []*Repo
	for repo := range s.conn.listAllRepositories(ctx) {
		r := githubRepoToRepo(repo, s.conn)
		r.Sources = append(r.Sources, externalServiceURN(s.svc))
		repos = append(repos, r)
	}
	return repos, nil
}

func githubRepoToRepo(ghrepo *github.Repository, conn *githubConnection) *Repo {
	return &Repo{
		Name:         string(githubRepositoryToRepoPath(conn, ghrepo)),
		CloneURL:     conn.authenticatedRemoteURL(ghrepo),
		ExternalRepo: *github.ExternalRepoSpec(ghrepo, *conn.baseURL),
		Description:  ghrepo.Description,
		Fork:         ghrepo.IsFork,
		Archived:     ghrepo.IsArchived,
	}
}

func externalServiceURN(svc *api.ExternalService) string {
	return "extsvc:" + strconv.FormatInt(svc.ID, 10)
}
