package resolvers

import (
	"context"
	"strings"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

const batchSpecExecutionIDKind = "BatchSpecExecution"

func marshalBatchSpecExecutionRandID(id string) graphql.ID {
	return relay.MarshalID(batchSpecExecutionIDKind, id)
}

func unmarshalBatchSpecExecutionRandID(id graphql.ID) (batchSpecExecutionID string, err error) {
	err = relay.UnmarshalSpec(id, &batchSpecExecutionID)
	return
}

type batchSpecExecutionResolver struct {
	store *store.Store
	exec  *btypes.BatchSpecExecution
}

// Type guard.
var _ graphqlbackend.BatchSpecExecutionResolver = &batchSpecExecutionResolver{}

func (r *batchSpecExecutionResolver) ID() graphql.ID {
	return marshalBatchSpecExecutionRandID(r.exec.RandID)
}

func (r *batchSpecExecutionResolver) InputSpec() string {
	return r.exec.BatchSpec
}

func (r *batchSpecExecutionResolver) State() string {
	return strings.ToUpper(string(r.exec.State))
}

func (r *batchSpecExecutionResolver) CreatedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: r.exec.CreatedAt}
}

func (r *batchSpecExecutionResolver) StartedAt() *graphqlbackend.DateTime {
	if r.exec.StartedAt == nil {
		return nil
	}
	return &graphqlbackend.DateTime{Time: *r.exec.StartedAt}
}

func (r *batchSpecExecutionResolver) FinishedAt() *graphqlbackend.DateTime {
	if r.exec.FinishedAt == nil {
		return nil
	}
	return &graphqlbackend.DateTime{Time: *r.exec.FinishedAt}
}

func (r *batchSpecExecutionResolver) Failure() *string {
	return r.exec.FailureMessage
}

func (r *batchSpecExecutionResolver) Steps() graphqlbackend.BatchSpecExecutionStepsResolver {
	return &batchSpecExecutionStepsResolver{
		store: r.store,
		exec:  r.exec,
	}
}

func (r *batchSpecExecutionResolver) PlaceInQueue() *int32 {
	// TODO(eseliger): Implement this.
	return nil
}

func (r *batchSpecExecutionResolver) BatchSpec(ctx context.Context) (graphqlbackend.BatchSpecResolver, error) {
	if r.exec.BatchSpecID == 0 {
		return nil, nil
	}
	spec, err := r.store.GetBatchSpec(ctx, store.GetBatchSpecOpts{ID: r.exec.BatchSpecID})
	if err != nil {
		return nil, err
	}
	return &batchSpecResolver{store: r.store, batchSpec: spec}, nil
}

func (r *batchSpecExecutionResolver) Initiator(ctx context.Context) (*graphqlbackend.UserResolver, error) {
	return graphqlbackend.UserByIDInt32(ctx, r.store.DB(), r.exec.UserID)
}

func (r *batchSpecExecutionResolver) Namespace(ctx context.Context) (*graphqlbackend.NamespaceResolver, error) {
	var (
		namespace graphqlbackend.NamespaceResolver
		err       error
	)
	if r.exec.NamespaceUserID != 0 {
		namespace.Namespace, err = graphqlbackend.UserByIDInt32(
			ctx,
			r.store.DB(),
			r.exec.NamespaceUserID,
		)
		return &namespace, err
	}
	namespace.Namespace, err = graphqlbackend.OrgByIDInt32(
		ctx,
		r.store.DB(),
		r.exec.NamespaceOrgID,
	)
	return &namespace, err
}

type batchSpecExecutionStepsResolver struct {
	store *store.Store
	exec  *btypes.BatchSpecExecution
}

var _ graphqlbackend.BatchSpecExecutionStepsResolver = &batchSpecExecutionStepsResolver{}

func (r *batchSpecExecutionStepsResolver) Setup() []graphqlbackend.ExecutionLogEntryResolver {
	return r.executionLogEntryResolversWithPrefix("setup.")
}

func (r *batchSpecExecutionStepsResolver) SrcPreview() graphqlbackend.ExecutionLogEntryResolver {
	if entry, ok := r.findExecutionLogEntry("step.src.0"); ok {
		return graphqlbackend.NewExecutionLogEntryResolver(r.store.DB(), entry)
	}

	return nil
}

func (r *batchSpecExecutionStepsResolver) Teardown() []graphqlbackend.ExecutionLogEntryResolver {
	return r.executionLogEntryResolversWithPrefix("teardown.")
}

func (r *batchSpecExecutionStepsResolver) findExecutionLogEntry(key string) (workerutil.ExecutionLogEntry, bool) {
	for _, entry := range r.exec.ExecutionLogs {
		if entry.Key == key {
			return entry, true
		}
	}

	return workerutil.ExecutionLogEntry{}, false
}

func (r *batchSpecExecutionStepsResolver) executionLogEntryResolversWithPrefix(prefix string) []graphqlbackend.ExecutionLogEntryResolver {
	var resolvers []graphqlbackend.ExecutionLogEntryResolver
	for _, entry := range r.exec.ExecutionLogs {
		if !strings.HasPrefix(entry.Key, prefix) {
			continue
		}
		r := graphqlbackend.NewExecutionLogEntryResolver(r.store.DB(), entry)
		resolvers = append(resolvers, r)
	}

	return resolvers
}
