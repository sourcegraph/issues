package resolvers

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/syncer"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
)

type changesetsConnectionResolver struct {
	store *store.Store

	opts store.ListChangesetsOpts
	// 🚨 SECURITY: If the given opts do not reveal hidden information about a
	// changeset by including the changeset in the result set, this should be
	// set to true.
	optsSafe bool

	once sync.Once
	// changesets contains all changesets in this connection.
	changesets btypes.Changesets
	next       int64
	err        error
}

func (r *changesetsConnectionResolver) Nodes(ctx context.Context) ([]graphqlbackend.ChangesetResolver, error) {
	changesetsPage, _, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}
	// 🚨 SECURITY: database.Repos.GetRepoIDsSet uses the authzFilter under the hood and
	// filters out repositories that the user doesn't have access to.
	reposByID, err := r.store.Repos().GetReposSetByIDs(ctx, changesetsPage.RepoIDs()...)
	if err != nil {
		return nil, err
	}

	scheduledSyncs := make(map[int64]time.Time)
	changesetIDs := changesetsPage.IDs()
	if len(changesetIDs) > 0 {
		syncData, err := r.store.ListChangesetSyncData(ctx, store.ListChangesetSyncDataOpts{ChangesetIDs: changesetIDs})
		if err != nil {
			return nil, err
		}
		for _, d := range syncData {
			scheduledSyncs[d.ChangesetID] = syncer.NextSync(r.store.Clock(), d)
		}
	}

	resolvers := make([]graphqlbackend.ChangesetResolver, 0, len(changesetsPage))
	for _, c := range changesetsPage {
		resolvers = append(resolvers, NewChangesetResolverWithNextSync(r.store, c, reposByID[c.RepoID], scheduledSyncs[c.ID]))
	}

	return resolvers, nil
}

func (r *changesetsConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	count, err := r.store.CountChangesets(ctx, store.CountChangesetsOpts{
		BatchChangeID:        r.opts.BatchChangeID,
		ExternalState:        r.opts.ExternalState,
		ExternalReviewState:  r.opts.ExternalReviewState,
		ExternalCheckState:   r.opts.ExternalCheckState,
		ReconcilerStates:     r.opts.ReconcilerStates,
		OwnedByBatchChangeID: r.opts.OwnedByBatchChangeID,
		PublicationState:     r.opts.PublicationState,
		TextSearch:           r.opts.TextSearch,
		EnforceAuthz:         !r.optsSafe,
		OnlyArchived:         r.opts.OnlyArchived,
		IncludeArchived:      r.opts.IncludeArchived,
	})
	return int32(count), err
}

// compute loads all changesets matched by r.opts.
// If r.optsSafe is true, it returns all of them. If not, it filters out the
// ones to which the user doesn't have access by using the authz filter.
func (r *changesetsConnectionResolver) compute(ctx context.Context) (cs btypes.Changesets, next int64, err error) {
	r.once.Do(func() {
		opts := r.opts
		if !r.optsSafe {
			opts.EnforceAuthz = true
		}
		r.changesets, r.next, r.err = r.store.ListChangesets(ctx, opts)
	})

	return r.changesets, r.next, r.err
}

func (r *changesetsConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	_, next, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}

	if next > 0 {
		return graphqlutil.NextPageCursor(strconv.Itoa(int(next))), nil
	}

	return graphqlutil.HasNextPage(false), nil
}
