package repos

import (
	"context"
	"sync"
	"time"

	"github.com/goware/urlx"
	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/phabricator"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/schema"
)

// A PhabricatorSource yields repositories from a single Phabricator connection configured
// in Sourcegraph via the external services configuration.
type PhabricatorSource struct {
	svc  *ExternalService
	conn *schema.PhabricatorConnection
	cf   *httpcli.Factory

	mu  sync.Mutex
	cli *phabricator.Client
}

// NewPhabricatorSource returns a new PhabricatorSource from the given external service.
func NewPhabricatorSource(svc *ExternalService, cf *httpcli.Factory) (*PhabricatorSource, error) {
	var c schema.PhabricatorConnection
	if err := jsonc.Unmarshal(svc.Config, &c); err != nil {
		return nil, errors.Wrapf(err, "external service id=%d config error", svc.ID)
	}
	return &PhabricatorSource{svc: svc, conn: &c, cf: cf}, nil
}

// ListRepos returns all Phabricator repositories accessible to all connections configured
// in Sourcegraph via the external services configuration.
func (s *PhabricatorSource) ListRepos(ctx context.Context, results chan SourceResult) {
	cli, err := s.client(ctx)
	if err != nil {
		results <- SourceResult{Source: s, Err: err}
		return
	}

	cursor := &phabricator.Cursor{Limit: 100, Order: "oldest"}
	for {
		var page []*phabricator.Repo
		page, cursor, err = cli.ListRepos(ctx, phabricator.ListReposArgs{Cursor: cursor})
		if err != nil {
			results <- SourceResult{Source: s, Err: err}
			return
		}

		for _, r := range page {
			if r.VCS != "git" || r.Status == "inactive" {
				continue
			}

			repo, err := s.makeRepo(r)
			if err != nil {
				results <- SourceResult{Source: s, Err: err}
				return
			}
			results <- SourceResult{Source: s, Repo: repo}
		}

		if cursor.After == "" {
			break
		}
	}
}

// ExternalServices returns a singleton slice containing the external service.
func (s *PhabricatorSource) ExternalServices() ExternalServices {
	return ExternalServices{s.svc}
}

func (s *PhabricatorSource) makeRepo(repo *phabricator.Repo) (*Repo, error) {
	var external []*phabricator.URI
	builtin := make(map[string]*phabricator.URI)

	for _, u := range repo.URIs {
		if u.Disabled || u.Normalized == "" {
			continue
		} else if u.BuiltinIdentifier != "" {
			builtin[u.BuiltinProtocol+"+"+u.BuiltinIdentifier] = u
		} else {
			external = append(external, u)
		}
	}

	var name string
	if len(external) > 0 {
		name = external[0].Normalized
	}

	var cloneURL string
	for _, alt := range [...]struct {
		protocol, identifier string
	}{ // Ordered by priority.
		{"https", "shortname"},
		{"https", "callsign"},
		{"https", "id"},
		{"ssh", "shortname"},
		{"ssh", "callsign"},
		{"ssh", "id"},
	} {
		if u, ok := builtin[alt.protocol+"+"+alt.identifier]; ok {
			cloneURL = u.Effective
			// TODO(tsenart): Authenticate the cloneURL with the user's
			// VCS password once we have that setting in the config. The
			// Conduit token can't be used for cloning.
			// cloneURL = setUserinfoBestEffort(cloneURL, conn.VCSPassword, "")

			if name == "" {
				name = u.Normalized
			}
		}
	}

	if cloneURL == "" {
		log15.Warn("unable to construct clone URL for repo", "name", name, "phabricator_id", repo.PHID)
	}

	if name == "" {
		return nil, errors.Errorf("no canonical name available for repo with id=%v", repo.PHID)
	}

	serviceID, err := urlx.NormalizeString(s.conn.Url)
	if err != nil {
		// Should never happen. URL must be validated on input.
		panic(err)
	}

	urn := s.svc.URN()
	return &Repo{
		Name: name,
		URI:  name,
		ExternalRepo: api.ExternalRepoSpec{
			ID:          repo.PHID,
			ServiceType: extsvc.TypePhabricator,
			ServiceID:   serviceID,
		},
		Sources: map[string]*SourceInfo{
			urn: {
				ID:       urn,
				CloneURL: cloneURL,
				// TODO(tsenart): We need a way for admins to specify which URI to
				// use as a CloneURL. Do they want to use https + shortname, git + callsign
				// an external URI that's mirrored or observed, etc.
				// This must be figured out when starting to integrate the new Syncer with this
				// source.
			},
		},
		Metadata: repo,
	}, nil
}

// client initialises the phabricator.Client if it isn't initialised yet.
// This is done lazily instead of in NewPhabricatorSource so that we have
// access to the context.Context passed in via ListRepos.
func (s *PhabricatorSource) client(ctx context.Context) (*phabricator.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cli != nil {
		return s.cli, nil
	}

	hc, err := s.cf.Doer()
	if err != nil {
		return nil, err
	}

	s.cli, err = phabricator.NewClient(ctx, s.conn.Url, s.conn.Token, hc)
	return s.cli, err
}

// RunPhabricatorRepositorySyncWorker runs the worker that syncs repositories from Phabricator to Sourcegraph
func RunPhabricatorRepositorySyncWorker(ctx context.Context, s Store) {
	cf := httpcli.NewExternalHTTPClientFactory()

	for {
		phabs, err := s.ListExternalServices(ctx, StoreListExternalServicesArgs{
			Kinds: []string{extsvc.KindPhabricator},
		})
		if err != nil {
			log15.Error("unable to fetch Phabricator connections", "err", err)
		}

		for _, phab := range phabs {
			src, err := NewPhabricatorSource(phab, cf)
			if err != nil {
				log15.Error("failed to instantiate PhabricatorSource", "err", err)
				continue
			}

			repos, err := listAll(ctx, src)
			if err != nil {
				log15.Error("Error fetching Phabricator repos", "err", err)
				continue
			}

			err = updatePhabRepos(ctx, repos)
			if err != nil {
				log15.Error("Error updating Phabricator repos", "err", err)
				continue
			}

			cfg, err := phab.Configuration()
			if err != nil {
				log15.Error("failed to parse Phabricator config", "err", err)
				continue
			}

			phabricatorUpdateTime.WithLabelValues(
				cfg.(*schema.PhabricatorConnection).Url,
			).Set(float64(time.Now().Unix()))
		}

		time.Sleep(ConfRepoListUpdateInterval())
	}
}

// updatePhabRepos ensures that all provided repositories exist in the phabricator_repos table.
func updatePhabRepos(ctx context.Context, repos []*Repo) error {
	for _, r := range repos {
		repo := r.Metadata.(*phabricator.Repo)
		err := api.InternalClient.PhabricatorRepoCreate(
			ctx,
			api.RepoName(r.Name),
			repo.Callsign,
			r.ExternalRepo.ServiceID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
