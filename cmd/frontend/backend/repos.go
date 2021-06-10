package backend

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbcache"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/inventory"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/types"
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

var Repos *repos

func InitRepos(db dbutil.DB) {
	rstore := database.Repos(db)
	Repos = &repos{
		store: rstore,
		cache: dbcache.NewDefaultRepoLister(rstore),
	}
}

type repos struct {
	store *database.RepoStore
	cache *dbcache.DefaultRepoLister
}

func (s *repos) Get(ctx context.Context, repo api.RepoID) (_ *types.Repo, err error) {
	if Mocks.Repos.Get != nil {
		return Mocks.Repos.Get(ctx, repo)
	}

	ctx, done := trace(ctx, "Repos", "Get", repo, &err)
	defer done()

	return s.store.Get(ctx, repo)
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

	switch repo, err := s.store.GetByName(ctx, name); {
	case err == nil:
		return repo, nil
	case !errcode.IsNotFound(err):
		return nil, err
	case envvar.SourcegraphDotComMode():
		// Automatically add repositories on Sourcegraph.com.
		newName, err := s.Add(ctx, name)
		if err != nil {
			return nil, err
		}
		return s.store.GetByName(ctx, newName)
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

var metricIsRepoCloneable = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "src_frontend_repo_add_is_cloneable",
	Help: "temporary metric to measure if this codepath is valuable on sourcegraph.com",
}, []string{"status"})

// Add adds the repository with the given name to the database by calling
// repo-updater when in sourcegraph.com mode. It's possible that the repo has
// been renamed on the code host in which case a different name may be returned.
func (s *repos) Add(ctx context.Context, name api.RepoName) (addedName api.RepoName, err error) {
	ctx, done := trace(ctx, "Repos", "Add", name, &err)
	defer done()

	// Avoid hitting repo-updater (and incurring a hit against our GitHub/etc. API rate
	// limit) for repositories that don't exist or private repositories that people attempt to
	// access.
	if host := extsvc.CodeHostOf(name, extsvc.PublicCodeHosts...); host != nil {
		status := "unknown"
		defer func() {
			metricIsRepoCloneable.WithLabelValues(status).Inc()
		}()

		if err := gitserver.DefaultClient.IsRepoCloneable(ctx, name); err != nil {
			if ctx.Err() != nil {
				status = "timeout"
			} else {
				status = "fail"
			}
			return "", err
		}
		status = "success"
	}

	// Looking up the repo in repo-updater makes it sync that repo to the
	// database on sourcegraph.com if that repo is from github.com or gitlab.com
	lookupResult, err := repoupdater.DefaultClient.RepoLookup(ctx, protocol.RepoLookupArgs{Repo: name})
	if lookupResult != nil && lookupResult.Repo != nil {
		return lookupResult.Repo.Name, err
	}
	return "", err
}

func (s *repos) List(ctx context.Context, opt database.ReposListOptions) (repos []*types.Repo, err error) {
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

	return s.store.List(ctx, opt)
}

// ListIndexable calls database.DefaultRepos.List, with tracing. It lists ALL
// default repos which could include private user added repos.
func (s *repos) ListIndexable(ctx context.Context) (repos []types.RepoName, err error) {
	ctx, done := trace(ctx, "Repos", "ListIndexable", nil, &err)
	defer func() {
		if err == nil {
			span := opentracing.SpanFromContext(ctx)
			span.LogFields(otlog.Int("result.len", len(repos)))
		}
		done()
	}()
	return s.cache.List(ctx)
}

// ListDefault calls database.DefaultRepos.ListPublic, with tracing.
// It lists all public default repos and also any private repos added by the
// current user.
func (s *repos) ListDefault(ctx context.Context) (repos []types.RepoName, err error) {
	ctx, done := trace(ctx, "Repos", "ListDefault", nil, &err)
	defer func() {
		if err == nil {
			span := opentracing.SpanFromContext(ctx)
			span.LogFields(otlog.Int("result.len", len(repos)))
		}
		done()
	}()

	span := opentracing.SpanFromContext(ctx)
	repos, err = s.cache.ListPublic(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "listing default public repos")
	}
	span.LogFields(otlog.Int("public.len", len(repos)))

	// For authenticated users we also want to include any private repos they may have added
	if a := actor.FromContext(ctx); a.IsAuthenticated() {
		privateRepos, err := s.store.ListRepoNames(ctx, database.ReposListOptions{
			UserID:      a.UID,
			OnlyPrivate: true,
		})
		if err != nil {
			return nil, errors.Wrap(err, "getting user private repos")
		}
		span.LogFields(otlog.Int("private.len", len(privateRepos)))
		repos = append(repos, privateRepos...)
	}

	return repos, nil
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

	invCtx, err := InventoryContext(repo.Name, commitID, forceEnhancedLanguageDetection)
	if err != nil {
		return nil, err
	}

	root, err := git.Stat(ctx, repo.Name, commitID, "")
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
