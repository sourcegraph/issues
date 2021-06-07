package resolvers

import (
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
)

type pendingBatchSpecResolver struct {
	pbs *types.PendingBatchSpec
}

var _ graphqlbackend.PendingBatchSpecResolver = &pendingBatchSpecResolver{}

func (r *pendingBatchSpecResolver) ID() graphql.ID {
	return marshalPendingBatchSpecID(r.pbs.ID)
}

func (r *pendingBatchSpecResolver) State() string {
	return r.pbs.State
}

const pendingBatchSpecIDKind = "PendingBatchSpec"

func marshalPendingBatchSpecID(id int64) graphql.ID {
	return relay.MarshalID(pendingBatchSpecIDKind, id)
}

// func unmarshalPendingBatchSpecID(id graphql.ID) (pendingBatchSpecID string, err error) {
// 	err = relay.UnmarshalSpec(id, &pendingBatchSpecID)
// 	return
// }
