package graphqlbackend

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

var cf = httpcli.NewExternalHTTPClientFactory()

type affiliatedRepositoriesConnection struct {
	userID   int32
	codeHost int64
	query    string

	once  sync.Once
	nodes []*codeHostRepositoryResolver
	err   error
	db    dbutil.DB
}

func (a *affiliatedRepositoriesConnection) Nodes(ctx context.Context) ([]*codeHostRepositoryResolver, error) {
	a.once.Do(func() {
		var (
			svcs []*types.ExternalService
			err  error
		)
		// get all external services for user, or for the specified external service
		if a.codeHost == 0 {
			svcs, err = database.ExternalServices(a.db).List(ctx, database.ExternalServicesListOptions{NamespaceUserID: a.userID})
			if err != nil {
				a.err = err
				return
			}
		} else {
			svc, err := database.ExternalServices(a.db).GetByID(ctx, a.codeHost)
			if err != nil {
				a.err = err
				return
			}
			// 🚨 SECURITY: if the user doesn't own this service, check they're site admin
			if svc.NamespaceUserID != a.userID {
				a.err = errors.New("external service must be owned by specified user")
				return
			}
			svcs = append(svcs, svc)
		}

		type affiliatedResult struct {
			svcID int64
			repos []types.CodeHostRepository
			err   error
		}

		// get Source for all external services
		var (
			results  = make(chan affiliatedResult, len(svcs))
			svcsByID = make(map[int64]*types.ExternalService)
			pending  int
		)

		store := repos.NewStore(a.db, sql.TxOptions{Isolation: sql.LevelDefault})
		{
			m := repos.NewStoreMetrics()
			m.MustRegister(prometheus.DefaultRegisterer)
			store.Metrics = m
		}

		for _, svc := range svcs {
			svcsByID[svc.ID] = svc
			src, err := repos.NewSource(svc, cf, store)
			if err != nil {
				a.err = err
				return
			}
			af, ok := src.(repos.AffiliatedRepositorySource)
			if !ok {
				continue
			}
			pending++
			svcID := svc.ID
			goroutine.Go(func() {
				affiliated, err := af.AffiliatedRepositories(ctx)
				results <- affiliatedResult{
					svcID: svcID,
					repos: affiliated,
					err:   err,
				}
			})
		}

		// are we allowed to show the user private repos?
		allowPrivate, err := allowPrivate(ctx, a.db, a.userID)
		if err != nil {
			a.err = err
			return
		}

		// collect all results
		var fetchErrors []error
		a.nodes = []*codeHostRepositoryResolver{}
		for i := 0; i < pending; i++ {
			select {
			case result := <-results:
				if result.err != nil {
					// An error from one code is not fatal
					log15.Error("getting affiliated repos", "externalServiceId", result.svcID, "err", err)
					fetchErrors = append(fetchErrors, result.err)
					continue
				}
				for _, repo := range result.repos {
					if a.query != "" && !strings.Contains(strings.ToLower(repo.Name), a.query) {
						continue
					}
					if !allowPrivate && repo.Private {
						continue
					}
					repo := repo
					a.nodes = append(a.nodes, &codeHostRepositoryResolver{
						db:       a.db,
						codeHost: svcsByID[repo.CodeHostID],
						repo:     &repo,
					})
				}
			case <-ctx.Done():
				a.err = ctx.Err()
				return
			}
		}

		sort.Slice(a.nodes, func(i, j int) bool {
			return a.nodes[i].repo.Name < a.nodes[j].repo.Name
		})

		if len(fetchErrors) == pending {
			// All hosts failed
			a.nodes = nil
			a.err = errors.New("failed to fetch from any code host")
		}
	})

	return a.nodes, a.err
}

type codeHostRepositoryResolver struct {
	repo     *types.CodeHostRepository
	codeHost *types.ExternalService
	db       dbutil.DB
}

func (r *codeHostRepositoryResolver) Name() string {
	return r.repo.Name
}

func (r *codeHostRepositoryResolver) Private() bool {
	return r.repo.Private
}

func (r *codeHostRepositoryResolver) CodeHost(ctx context.Context) *externalServiceResolver {
	return &externalServiceResolver{
		db:              r.db,
		externalService: r.codeHost,
	}
}

func allowPrivate(ctx context.Context, db dbutil.DB, userID int32) (bool, error) {
	mode, err := database.Users(db).UserAllowedExternalServices(ctx, userID)
	if err != nil {
		return false, err
	}
	return mode == conf.ExternalServiceModeAll, nil
}
