package backend

import (
	"context"
	"net/url"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
)

// CachedGitRepo returns a handle to the Git repository that does not know the remote URL. If
// knowing the remote URL is necessary to perform any operations (from method calls on the return
// value), those operations will fail. This occurs when the repository isn't cloned on gitserver or
// when an update is needed (eg in ResolveRevision).
func CachedGitRepo(ctx context.Context, repo *types.Repo) (*gitserver.Repo, error) {
	r, err := quickGitserverRepo(ctx, repo.Name, repo.ExternalRepo.ServiceType)
	if err != nil {
		return nil, err
	}
	if r != nil {
		return r, nil
	}
	return &gitserver.Repo{Name: repo.Name}, nil
}

// GitRepo returns a handle to the Git repository with the up-to-date (as of the time of this call)
// remote URL. See CachedGitRepo for when this is necessary vs. unnecessary.
func GitRepo(ctx context.Context, repo *types.Repo) (gitserver.Repo, error) {
	gitserverRepo, err := quickGitserverRepo(ctx, repo.Name, repo.ExternalRepo.ServiceType)
	if err != nil {
		return gitserver.Repo{Name: repo.Name}, err
	}
	if gitserverRepo != nil {
		return *gitserverRepo, nil
	}

	result, err := repoupdater.DefaultClient.RepoLookup(ctx, protocol.RepoLookupArgs{
		Repo: repo.Name,
	})
	if err != nil {
		return gitserver.Repo{Name: repo.Name}, err
	}
	if result.Repo == nil {
		return gitserver.Repo{Name: repo.Name}, &repoupdater.ErrNotFound{Repo: repo.Name, IsNotFound: true}
	}
	return gitserver.Repo{Name: result.Repo.Name, URL: result.Repo.VCS.URL}, nil
}

func quickGitserverRepo(ctx context.Context, repo api.RepoName, serviceType string) (*gitserver.Repo, error) {
	// If it is possible to 100% correctly determine it statically, use a fast path. This is
	// used to avoid a RepoLookup call for public GitHub.com and GitLab.com repositories
	// (especially on Sourcegraph.com), which reduces rate limit pressure significantly.
	//
	// This fails for private repositories, which require authentication in the URL userinfo.

	lowerRepo := strings.ToLower(string(repo))
	var hasToken func(context.Context) (bool, error)
	switch {
	case serviceType == extsvc.TypeGitHub && strings.HasPrefix(lowerRepo, "github.com/"):
		hasToken = hasGitHubDotComToken
	case serviceType == extsvc.TypeGitLab && strings.HasPrefix(lowerRepo, "gitlab.com/"):
		hasToken = hasGitLabDotComToken
	default:
		return nil, nil
	}

	r := &gitserver.Repo{Name: repo, URL: "https://" + string(repo) + ".git"}
	if envvar.SourcegraphDotComMode() {
		return r, nil
	}

	ok, err := hasToken(ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		return r, nil
	}

	// Fall back to performing full RepoLookup, which will hit the code host.
	return nil, nil
}

// hasGitHubDotComToken reports whether there are any personal access tokens configured for
// github.com.
func hasGitHubDotComToken(ctx context.Context) (hasToken bool, _ error) {
	opt := db.ExternalServicesListOptions{
		Kinds: []string{extsvc.KindGitHub},
		LimitOffset: &db.LimitOffset{
			Limit: 500, // The number is randomly chosen
		},
	}
	for {
		svcs, err := db.ExternalServices.List(ctx, opt)
		if err != nil {
			return false, errors.Wrap(err, "list")
		}
		if len(svcs) == 0 {
			break // No more results, exiting
		}
		opt.AfterID = svcs[len(svcs)-1].ID // Advance the cursor

		for _, svc := range svcs {
			cfg, err := extsvc.ParseConfig(svc.Kind, svc.Config)
			if err != nil {
				return false, errors.Wrap(err, "parse config")
			}

			var conn *schema.GitHubConnection
			switch c := cfg.(type) {
			case *schema.GitHubConnection:
				conn = c
			default:
				log15.Error("hasGitHubDotComToken", "error", errors.Errorf("want *schema.GitHubConnection but got %T", cfg))
				continue
			}

			u, err := url.Parse(conn.Url)
			if err != nil {
				continue
			}
			hostname := strings.ToLower(u.Hostname())
			if (hostname == "github.com" || hostname == "api.github.com") && conn.Token != "" {
				return true, nil
			}
		}

		if len(svcs) < opt.Limit {
			break // Less results than limit means we've reached end
		}
	}
	return false, nil
}

// hasGitLabDotComToken reports whether there are any personal access tokens configured for
// github.com.
func hasGitLabDotComToken(ctx context.Context) (bool, error) {
	opt := db.ExternalServicesListOptions{
		Kinds: []string{extsvc.KindGitLab},
		LimitOffset: &db.LimitOffset{
			Limit: 500, // The number is randomly chosen
		},
	}
	for {
		svcs, err := db.ExternalServices.List(ctx, opt)
		if err != nil {
			return false, errors.Wrap(err, "list")
		}
		if len(svcs) == 0 {
			break // No more results, exiting
		}
		opt.AfterID = svcs[len(svcs)-1].ID // Advance the cursor

		for _, svc := range svcs {
			cfg, err := extsvc.ParseConfig(svc.Kind, svc.Config)
			if err != nil {
				return false, errors.Wrap(err, "parse config")
			}

			var conn *schema.GitLabConnection
			switch c := cfg.(type) {
			case *schema.GitLabConnection:
				conn = c
			default:
				log15.Error("hasGitLabDotComToken", "error", errors.Errorf("want *schema.GitLabConnection but got %T", cfg))
				continue
			}

			u, err := url.Parse(conn.Url)
			if err != nil {
				continue
			}
			hostname := strings.ToLower(u.Hostname())
			if hostname == "gitlab.com" && conn.Token != "" {
				return true, nil
			}
		}

		if len(svcs) < opt.Limit {
			break // Less results than limit means we've reached end
		}
	}
	return false, nil
}

// ResolveRev will return the absolute commit for a commit-ish spec in a repo.
// If no rev is specified, HEAD is used.
// Error cases:
// * Repo does not exist: vcs.RepoNotExistError
// * Commit does not exist: git.RevisionNotFoundError
// * Empty repository: git.RevisionNotFoundError
// * The user does not have permission: errcode.IsNotFound
// * Other unexpected errors.
func (s *repos) ResolveRev(ctx context.Context, repo *types.Repo, rev string) (commitID api.CommitID, err error) {
	if Mocks.Repos.ResolveRev != nil {
		return Mocks.Repos.ResolveRev(ctx, repo, rev)
	}

	ctx, done := trace(ctx, "Repos", "ResolveRev", map[string]interface{}{"repo": repo.Name, "rev": rev}, &err)
	defer done()

	// We start out by using a CachedGitRepo which doesn't have a remote URL.
	// If we need the remote URL, git.ResolveRevision will ask for it via
	// remoteURLFunc (which is costly as it e.g. consumes code host API
	// requests).
	gitserverRepo, err := CachedGitRepo(ctx, repo)
	maybeLogRepoUpdaterError(repo, err)
	if err != nil && !isIgnorableRepoUpdaterError(err) {
		return "", err
	}
	remoteURLFunc := func() (string, error) {
		grepo, err := GitRepo(ctx, repo)
		if err != nil {
			return "", err
		}
		return grepo.URL, nil
	}
	return git.ResolveRevision(ctx, *gitserverRepo, remoteURLFunc, rev, git.ResolveRevisionOptions{})
}

func (s *repos) GetCommit(ctx context.Context, repo *types.Repo, commitID api.CommitID) (res *git.Commit, err error) {
	if Mocks.Repos.GetCommit != nil {
		return Mocks.Repos.GetCommit(ctx, repo, commitID)
	}

	ctx, done := trace(ctx, "Repos", "GetCommit", map[string]interface{}{"repo": repo.Name, "commitID": commitID}, &err)
	defer done()

	log15.Debug("svc.local.repos.GetCommit", "repo", repo.Name, "commitID", commitID)

	if !git.IsAbsoluteRevision(string(commitID)) {
		return nil, errors.Errorf("non-absolute CommitID for Repos.GetCommit: %v", commitID)
	}

	// We start out by using a CachedGitRepo which doesn't have a remote URL.
	// If we need the remote URL, git.ResolveRevision will ask for it via
	// remoteURLFunc (which is costly as it e.g. consumes code host API
	// requests).
	gitserverRepo, err := CachedGitRepo(ctx, repo)
	maybeLogRepoUpdaterError(repo, err)
	if err != nil && !isIgnorableRepoUpdaterError(err) {
		return nil, err
	}
	remoteURLFunc := func() (string, error) {
		grepo, err := GitRepo(ctx, repo)
		if err != nil {
			return "", err
		}
		return grepo.URL, nil
	}
	return git.GetCommit(ctx, *gitserverRepo, remoteURLFunc, commitID, git.ResolveRevisionOptions{})
}

func isIgnorableRepoUpdaterError(err error) bool {
	return errcode.IsNotFound(err) || errcode.IsUnauthorized(err) || errcode.IsTemporary(err)
}

func maybeLogRepoUpdaterError(repo *types.Repo, err error) {
	var msg string
	switch {
	case errcode.IsNotFound(err):
		msg = "Repository host reported a repository as not found. If this repository was deleted on its origin, the site admin must explicitly delete it on Sourcegraph."
	case errcode.IsUnauthorized(err):
		msg = "Repository host rejected as unauthorized an attempt to retrieve a repository's metadata. Check the repository host credentials in site configuration."
	case errcode.IsTemporary(err):
		msg = "Repository host was temporarily unavailable while retrieving repository information."
	}
	if msg != "" {
		log15.Warn(msg+" Consult repo-updater logs for more information.", "repo", repo.Name)
	}
}
