package resolvers

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

const bulkOperationIDKind = "BulkOperation"

func marshalBulkOperationID(id string) graphql.ID {
	return relay.MarshalID(bulkOperationIDKind, id)
}

func unmarshalBulkOperationID(id graphql.ID) (bulkOperationID string, err error) {
	err = relay.UnmarshalSpec(id, &bulkOperationID)
	return
}

type bulkOperationResolver struct {
	store         *store.Store
	bulkOperation *btypes.BulkOperation
}

var _ graphqlbackend.BulkOperationResolver = &bulkOperationResolver{}

func (r *bulkOperationResolver) ID() graphql.ID {
	return marshalBulkOperationID(r.bulkOperation.ID)
}

func (r *bulkOperationResolver) Type() (string, error) {
	return changesetJobTypeToBulkOperationType(r.bulkOperation.Type)
}

func (r *bulkOperationResolver) State() string {
	return string(r.bulkOperation.State)
}

func (r *bulkOperationResolver) Progress() float64 {
	return r.bulkOperation.Progress
}

func (r *bulkOperationResolver) Errors(ctx context.Context) ([]graphqlbackend.ChangesetJobErrorResolver, error) {
	errors, err := r.store.ListBulkOperationErrors(ctx, store.ListBulkOperationErrorsOpts{BulkOperationID: r.bulkOperation.ID})
	if err != nil {
		return nil, err
	}

	changesetIDs := uniqueChangesetIDsForBulkOperationErrors(errors)

	changesetsByID := map[int64]*btypes.Changeset{}
	reposByID := map[api.RepoID]*types.Repo{}
	if len(changesetIDs) > 0 {
		// Load all changesets and repos at once, to avoid N+1 queries.
		changesets, _, err := r.store.ListChangesets(ctx, store.ListChangesetsOpts{IDs: changesetIDs})
		if err != nil {
			return nil, err
		}
		for _, ch := range changesets {
			changesetsByID[ch.ID] = ch
		}
		// 🚨 SECURITY: database.Repos.GetReposSetByIDs uses the authzFilter under the hood and
		// filters out repositories that the user doesn't have access to.
		reposByID, err = r.store.Repos().GetReposSetByIDs(ctx, changesets.RepoIDs()...)
		if err != nil {
			return nil, err
		}
	}

	res := make([]graphqlbackend.ChangesetJobErrorResolver, 0, len(errors))
	for _, e := range errors {
		ch := changesetsByID[e.ChangesetID]
		repo, accessible := reposByID[ch.RepoID]
		resolver := &changesetJobErrorResolver{store: r.store, changeset: ch, repo: repo}
		if accessible {
			resolver.error = e.Error
		}
		res = append(res, resolver)
	}
	return res, nil
}

func (r *bulkOperationResolver) Initiator(ctx context.Context) (*graphqlbackend.UserResolver, error) {
	return graphqlbackend.UserByIDInt32(ctx, r.store.DB(), r.bulkOperation.UserID)
}

func (r *bulkOperationResolver) ChangesetCount() int32 {
	return r.bulkOperation.ChangesetCount
}

func (r *bulkOperationResolver) CreatedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: r.bulkOperation.CreatedAt}
}

func (r *bulkOperationResolver) FinishedAt() *graphqlbackend.DateTime {
	if r.bulkOperation.FinishedAt.IsZero() {
		return nil
	}
	return &graphqlbackend.DateTime{Time: r.bulkOperation.FinishedAt}
}

func changesetJobTypeToBulkOperationType(t btypes.ChangesetJobType) (string, error) {
	switch t {
	case btypes.ChangesetJobTypeComment:
		return "COMMENT", nil
	case btypes.ChangesetJobTypeDetach:
		return "DETACH", nil
	case btypes.ChangesetJobTypeReenqueue:
		return "REENQUEUE", nil
	case btypes.ChangesetJobTypeMerge:
		return "MERGE", nil
	case btypes.ChangesetJobTypeClose:
		return "CLOSE", nil
	case btypes.ChangesetJobTypePublish:
		return "PUBLISH", nil
	default:
		return "", errors.Errorf("invalid job type %q", t)
	}
}

func uniqueChangesetIDsForBulkOperationErrors(errors []*btypes.BulkOperationError) []int64 {
	changesetIDsMap := map[int64]struct{}{}
	changesetIDs := []int64{}
	for _, e := range errors {
		if _, ok := changesetIDsMap[e.ChangesetID]; ok {
			continue
		}
		changesetIDs = append(changesetIDs, e.ChangesetID)
		changesetIDsMap[e.ChangesetID] = struct{}{}
	}
	return changesetIDs
}
