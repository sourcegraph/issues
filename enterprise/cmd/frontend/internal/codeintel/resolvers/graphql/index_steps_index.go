package graphql

import (
	gql "github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	store "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type indexStepResolver struct {
	index store.Index
	entry *workerutil.ExecutionLogEntry
}

var _ gql.IndexStepResolver = &indexStepResolver{}

func (r *indexStepResolver) IndexerArgs() []string { return r.index.IndexerArgs }
func (r *indexStepResolver) Outfile() *string      { return strPtr(r.index.Outfile) }

func (r *indexStepResolver) LogEntry() gql.ExecutionLogEntryResolver {
	if r.entry != nil {
		return gql.NewExecutionLogEntryResolver(dbconn.Global, *r.entry)
	}

	return nil
}
