package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/schema"
)

type phabricator struct{}

type errPhabricatorRepoNotFound struct {
	args []interface{}
}

func (err errPhabricatorRepoNotFound) Error() string {
	return fmt.Sprintf("phabricator repo not found: %v", err.args)
}

func (err errPhabricatorRepoNotFound) NotFound() bool { return true }

func (*phabricator) Create(ctx context.Context, callsign string, name api.RepoName, phabURL string) (*types.PhabricatorRepo, error) {
	r := &types.PhabricatorRepo{
		Callsign: callsign,
		Name:     name,
		URL:      phabURL,
	}
	err := dbconn.Global.QueryRowContext(
		ctx,
		"INSERT INTO phabricator_repos(callsign, repo_name, url) VALUES($1, $2, $3) RETURNING id",
		r.Callsign, r.Name, r.URL).Scan(&r.ID)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (p *phabricator) CreateOrUpdate(ctx context.Context, callsign string, name api.RepoName, phabURL string) (*types.PhabricatorRepo, error) {
	r := &types.PhabricatorRepo{
		Callsign: callsign,
		Name:     name,
		URL:      phabURL,
	}
	err := dbconn.Global.QueryRowContext(
		ctx,
		"UPDATE phabricator_repos SET callsign=$1, url=$2, updated_at=now() WHERE repo_name=$3 RETURNING id",
		r.Callsign, r.URL, r.Name).Scan(&r.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return p.Create(ctx, callsign, name, phabURL)
		}
		return nil, err
	}
	return r, nil
}

func (p *phabricator) CreateIfNotExists(ctx context.Context, callsign string, name api.RepoName, phabURL string) (*types.PhabricatorRepo, error) {
	repo, err := p.GetByName(ctx, name)
	if err != nil {
		if _, ok := err.(errPhabricatorRepoNotFound); !ok {
			return nil, err
		}
		return p.Create(ctx, callsign, name, phabURL)
	}
	return repo, nil
}

func (*phabricator) getBySQL(ctx context.Context, query string, args ...interface{}) ([]*types.PhabricatorRepo, error) {
	rows, err := dbconn.Global.QueryContext(ctx, "SELECT id, callsign, repo_name, url FROM phabricator_repos "+query, args...)
	if err != nil {
		return nil, err
	}

	repos := []*types.PhabricatorRepo{}
	defer rows.Close()
	for rows.Next() {
		r := types.PhabricatorRepo{}
		err := rows.Scan(&r.ID, &r.Callsign, &r.Name, &r.URL)
		if err != nil {
			return nil, err
		}
		repos = append(repos, &r)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return repos, nil
}

func (p *phabricator) getOneBySQL(ctx context.Context, query string, args ...interface{}) (*types.PhabricatorRepo, error) {
	rows, err := p.getBySQL(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		return nil, errPhabricatorRepoNotFound{args}
	}
	return rows[0], nil
}

func (p *phabricator) GetByName(ctx context.Context, name api.RepoName) (*types.PhabricatorRepo, error) {
	if Mocks.Phabricator.GetByName != nil {
		return Mocks.Phabricator.GetByName(name)
	}

	opt := ExternalServicesListOptions{
		Kinds: []string{extsvc.KindPhabricator},
		LimitOffset: &LimitOffset{
			Limit: 500, // The number is randomly chosen
		},
	}
	for {
		svcs, err := ExternalServices.List(ctx, opt)
		if err != nil {
			return nil, errors.Wrap(err, "list")
		}
		if len(svcs) == 0 {
			break // No more results, exiting
		}
		opt.AfterID = svcs[len(svcs)-1].ID // Advance the cursor

		for _, svc := range svcs {
			cfg, err := extsvc.ParseConfig(svc.Kind, svc.Config)
			if err != nil {
				return nil, errors.Wrap(err, "parse config")
			}

			var conn *schema.PhabricatorConnection
			switch c := cfg.(type) {
			case *schema.PhabricatorConnection:
				conn = c
			default:
				log15.Error("phabricator.GetByName", "error", errors.Errorf("want *schema.PhabricatorConnection but got %T", cfg))
				continue
			}

			for _, repo := range conn.Repos {
				if api.RepoName(repo.Path) == name {
					return &types.PhabricatorRepo{
						Name:     api.RepoName(repo.Path),
						Callsign: repo.Callsign,
						URL:      conn.Url,
					}, nil
				}
			}
		}

		if len(svcs) < opt.Limit {
			break // Less results than limit means we've reached end
		}
	}

	return p.getOneBySQL(ctx, "WHERE repo_name=$1", name)
}

type MockPhabricator struct {
	GetByName func(repo api.RepoName) (*types.PhabricatorRepo, error)
}
