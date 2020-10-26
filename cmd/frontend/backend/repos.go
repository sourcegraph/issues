package backend

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/inventory"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

// ErrRepoSeeOther indicates that the repo does not exist on this server but might exist on an external Sourcegraph
// server.
type ErrRepoSeeOther struct {
	// RedirectURL is the base URL for the repository at an external location.
	RedirectURL string
}

func (e ErrRepoSeeOther) Error() string {
	return fmt.Sprintf("repo not found at this location, but might exist at %s", e.RedirectURL)
}

var Repos = &repos{}

type repos struct{}

func (s *repos) Get(ctx context.Context, repo api.RepoID) (_ *types.Repo, err error) {
	if Mocks.Repos.Get != nil {
		return Mocks.Repos.Get(ctx, repo)
	}

	ctx, done := trace(ctx, "Repos", "Get", repo, &err)
	defer done()

	return db.Repos.Get(ctx, repo)
}

// GetByName retrieves the repository with the given name. On sourcegraph.com,
// if the name refers to a repository on a github.com or gitlab.com that is not
// yet present in the database, it will automatically look up the
// repository externally and add it to the database before returning it.
func (s *repos) GetByName(ctx context.Context, name api.RepoName) (_ *types.Repo, err error) {
	if Mocks.Repos.GetByName != nil {
		return Mocks.Repos.GetByName(ctx, name)
	}

	ctx, done := trace(ctx, "Repos", "GetByName", name, &err)
	defer done()

	switch repo, err := db.Repos.GetByName(ctx, name); {
	case err == nil:
		return repo, nil
	case !errcode.IsNotFound(err):
		return nil, err
	case envvar.SourcegraphDotComMode():
		// Automatically add repositories on Sourcegraph.com.
		if err := s.Add(ctx, name); err != nil {
			return nil, err
		}
		return db.Repos.GetByName(ctx, name)
	case shouldRedirect(name):
		return nil, ErrRepoSeeOther{RedirectURL: (&url.URL{
			Scheme:   "https",
			Host:     "sourcegraph.com",
			Path:     string(name),
			RawQuery: url.Values{"utm_source": []string{conf.DeployType()}}.Encode(),
		}).String()}
	default:
		return nil, err
	}
}

func shouldRedirect(name api.RepoName) bool {
	return !conf.Get().DisablePublicRepoRedirects &&
		extsvc.CodeHostOf(name, extsvc.PublicCodeHosts...) != nil
}

// Add adds the repository with the given name to the database by calling
// repo-updater when in sourcegraph.com mode.
func (s *repos) Add(ctx context.Context, name api.RepoName) (err error) {
	ctx, done := trace(ctx, "Repos", "Add", name, &err)
	defer done()

	// Avoid hitting repo-updater (and incurring a hit against our GitHub/etc. API rate
	// limit) for repositories that don't exist or private repositories that people attempt to
	// access.
	if host := extsvc.CodeHostOf(name, extsvc.PublicCodeHosts...); host != nil {
		gitserverRepo, err := quickGitserverRepo(ctx, name, host.ServiceType)
		if err != nil {
			return err
		}

		if gitserverRepo != nil {
			if err := gitserver.DefaultClient.IsRepoCloneable(ctx, *gitserverRepo); err != nil {
				return err
			}
		}
	}

	// Looking up the repo in repo-updater makes it sync that repo to the
	// database on sourcegraph.com if that repo is from github.com or gitlab.com
	_, err = repoupdater.DefaultClient.RepoLookup(ctx, protocol.RepoLookupArgs{Repo: name})
	return err
}

func (s *repos) List(ctx context.Context, opt db.ReposListOptions) (repos []*types.Repo, err error) {
	if Mocks.Repos.List != nil {
		return Mocks.Repos.List(ctx, opt)
	}

	ctx, done := trace(ctx, "Repos", "List", opt, &err)
	defer func() {
		if err == nil {
			span := opentracing.SpanFromContext(ctx)
			span.LogFields(otlog.Int("result.len", len(repos)))
		}
		done()
	}()

	return db.Repos.List(ctx, opt)
}

// ListDefault calls db.DefaultRepos.List, with tracing.
func (s *repos) ListDefault(ctx context.Context) (repos []*types.Repo, err error) {
	ctx, done := trace(ctx, "Repos", "ListDefault", nil, &err)
	defer func() {
		if err == nil {
			span := opentracing.SpanFromContext(ctx)
			span.LogFields(otlog.Int("result.len", len(repos)))
		}
		done()
	}()
	return db.DefaultRepos.List(ctx)
}

func (s *repos) GetInventory(ctx context.Context, repo *types.Repo, commitID api.CommitID, forceEnhancedLanguageDetection bool) (res *inventory.Inventory, err error) {
	if Mocks.Repos.GetInventory != nil {
		return Mocks.Repos.GetInventory(ctx, repo, commitID)
	}

	ctx, done := trace(ctx, "Repos", "GetInventory", map[string]interface{}{"repo": repo.Name, "commitID": commitID}, &err)
	defer done()

	// Cap GetInventory operation to some reasonable time.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	cachedRepo, err := CachedGitRepo(ctx, repo)
	if err != nil {
		return nil, err
	}

	invCtx, err := InventoryContext(*cachedRepo, commitID, forceEnhancedLanguageDetection)
	if err != nil {
		return nil, err
	}

	root, err := git.Stat(ctx, *cachedRepo, commitID, "")
	if err != nil {
		return nil, err
	}

	// In computing the inventory, sub-tree inventories are cached based on the OID of the Git
	// tree. Compared to per-blob caching, this creates many fewer cache entries, which means fewer
	// stores, fewer lookups, and less cache storage overhead. Compared to per-commit caching, this
	// yields a higher cache hit rate because most trees are unchanged across commits.
	inv, err := invCtx.Entries(ctx, root)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}
