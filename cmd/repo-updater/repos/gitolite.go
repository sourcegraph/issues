package repos

import (
	"context"
	"net/http"
	"strings"

	"github.com/sourcegraph/sourcegraph/internal/extsvc"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitolite"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/schema"
	"golang.org/x/sync/semaphore"
)

// A GitoliteSource yields repositories from a single Gitolite connection configured
// in Sourcegraph via the external services configuration.
type GitoliteSource struct {
	svc  *ExternalService
	conn *schema.GitoliteConnection
	// We ask gitserver to talk to gitolite because it holds the ssh keys
	// required for authentication.
	cli     *gitserver.Client
	exclude excludeFunc
}

// NewGitoliteSource returns a new GitoliteSource from the given external service.
func NewGitoliteSource(svc *ExternalService, cf *httpcli.Factory) (*GitoliteSource, error) {
	var c schema.GitoliteConnection
	if err := jsonc.Unmarshal(svc.Config, &c); err != nil {
		return nil, errors.Wrapf(err, "external service id=%d config error", svc.ID)
	}

	hc, err := cf.Doer(func(c *http.Client) error {
		if tr, ok := c.Transport.(*http.Transport); ok {
			tr.MaxIdleConnsPerHost = 500
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var eb excludeBuilder
	for _, r := range c.Exclude {
		eb.Exact(r.Name)
		eb.Pattern(r.Pattern)
	}
	exclude, err := eb.Build()
	if err != nil {
		return nil, err
	}

	return &GitoliteSource{
		svc:     svc,
		conn:    &c,
		cli:     gitserver.NewClient(hc),
		exclude: exclude,
	}, nil
}

// ListRepos returns all Gitolite repositories accessible to all connections configured
// in Sourcegraph via the external services configuration.
func (s *GitoliteSource) ListRepos(ctx context.Context, results chan SourceResult) {
	all, err := s.cli.ListGitolite(ctx, s.conn.Host)
	if err != nil {
		results <- SourceResult{Source: s, Err: err}
		return
	}

	for _, r := range all {
		repo := s.makeRepo(r)
		if !s.excludes(r, repo) {
			results <- SourceResult{Source: s, Repo: repo}
		}
	}
}

// ExternalServices returns a singleton slice containing the external service.
func (s GitoliteSource) ExternalServices() ExternalServices {
	return ExternalServices{s.svc}
}

func (s GitoliteSource) excludes(gr *gitolite.Repo, r *Repo) bool {
	return s.exclude(gr.Name) ||
		strings.ContainsAny(r.Name, "\\^$|()[]*?{},")
}

func (s GitoliteSource) makeRepo(repo *gitolite.Repo) *Repo {
	urn := s.svc.URN()
	name := string(reposource.GitoliteRepoName(s.conn.Prefix, repo.Name))
	return &Repo{
		Name:         name,
		URI:          name,
		ExternalRepo: gitolite.ExternalRepoSpec(repo, gitolite.ServiceID(s.conn.Host)),
		Sources: map[string]*SourceInfo{
			urn: {
				ID:       urn,
				CloneURL: repo.URL,
			},
		},
		Metadata: repo,
	}
}

// GitolitePhabricatorMetadataSyncer creates Phabricator repos (in the phabricator_repo table) for each Gitolite
// repo provided in it's Sync method. This is to satisfy the contract established by the "phabricator" setting in the
// Gitolite external service configuration.
//
// TODO(tsenart): This is a HUGE hack, but it lives to see another day. Erradicating this technical debt
// involves lifting the Phabricator integration to a first class citizen, so that it can be treated as source of
// truth for repos to be mirrored. This would allow using a Phabricator integration that observes another code host,
// like Gitolite, and provides URIs to those external code hosts that git-server can use as clone URLs, while
// repo links can still be the built-in Phabricator ones, as is usually expected by customers that rely on code
// intelligence. With a Phabricator integration similar to all other code hosts, we could remove all of the special code
// paths for Phabricator everywhere as well as the `phabricator_repo` table.
type GitolitePhabricatorMetadataSyncer struct {
	sem     *semaphore.Weighted // Only one sync at a time, like it was done before.
	counter int64               // Only sync every 10th time, like it was done before.
	store   Store               // Use to load the external services that yielded a given repo.
}

// NewGitolitePhabricatorMetadataSyncer returns a GitolitePhabricatorMetadataSyncer with
// the given parameters.
func NewGitolitePhabricatorMetadataSyncer(s Store) *GitolitePhabricatorMetadataSyncer {
	return &GitolitePhabricatorMetadataSyncer{
		sem:     semaphore.NewWeighted(1),
		counter: -1,
		store:   s,
	}
}

// Sync creates Phabricator repos for each of the given Gitolite repos.
// If this is confusing to you, that's because it is. Read the comment on
// the GitolitePhabricatorMetadataSyncer type.
func (s *GitolitePhabricatorMetadataSyncer) Sync(ctx context.Context, repos []*Repo) error {
	if !s.sem.TryAcquire(1) {
		log15.Info("existing gitolite/phabricator repo task still running, skipping")
		return nil
	}
	defer s.sem.Release(1)

	if s.counter++; s.counter%10 != 0 { // Only run every ten times.
		log15.Debug("phabricator metadata sync only runs every 10th gitolite sync. skipping", "counter", s.counter)
		return nil
	}

	// Group repos by external service so that we look-up external services from the DB
	// only once.
	var ids []int64
	grouped := make(map[int64]Repos)
	for _, r := range repos {
		if r.ExternalRepo.ServiceType != extsvc.TypeGitolite || r.IsDeleted() {
			continue
		}

		for _, id := range r.ExternalServiceIDs() {
			ids = append(ids, id)
			grouped[id] = append(grouped[id], r)
		}
	}

	if len(ids) == 0 {
		log15.Debug("phabricator metadata: nothing to sync")
		return nil
	}

	es, err := s.store.ListExternalServices(ctx, StoreListExternalServicesArgs{IDs: ids})
	if err != nil {
		return errors.Wrap(err, "gitolite-phabricator-metadata-syncer.store.list-external-services")
	}

	for _, e := range es {
		urn := e.URN()

		c, err := e.Configuration()
		if err != nil {
			return errors.Wrapf(err, "gitolite-phabricator-metadata-syncer.external-service-config: %s", urn)
		}

		conf := c.(*schema.GitoliteConnection)
		if conf.Phabricator == nil {
			log15.Warn("missing phabricator setting. skipping", "external-service", urn)
			continue
		}

		for _, r := range grouped[e.ID] {
			name := api.RepoName(r.Name)

			metadata, err := gitserver.DefaultClient.GetGitolitePhabricatorMetadata(ctx, conf.Host, name)
			if err != nil {
				log15.Warn("could not fetch valid Phabricator metadata for Gitolite repository. skipping.", "repo", name, "error", err)
				continue
			}

			if metadata.Callsign == "" {
				log15.Warn("empty Phabricator callsign for Gitolite repository. skipping.", "repo", name, "error", err)
				continue
			}

			if err := api.InternalClient.PhabricatorRepoCreate(ctx, name, metadata.Callsign, conf.Phabricator.Url); err != nil {
				log15.Warn("could not ensure Gitolite Phabricator mapping", "repo", name, "error", err)
			}
		}
	}

	log15.Info("updated gitolite/phabricator metadata for repos", "repos", len(repos))
	return nil
}
