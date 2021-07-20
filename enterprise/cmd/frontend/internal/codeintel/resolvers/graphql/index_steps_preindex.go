package graphql

import (
	gql "github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	store "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type preIndexStepResolver struct {
	step  store.DockerStep
	entry *workerutil.ExecutionLogEntry
}

var _ gql.PreIndexStepResolver = &preIndexStepResolver{}

func (r *preIndexStepResolver) Root() string       { return r.step.Root }
func (r *preIndexStepResolver) Image() string      { return r.step.Image }
func (r *preIndexStepResolver) Commands() []string { return r.step.Commands }

func (r *preIndexStepResolver) LogEntry() gql.ExecutionLogEntryResolver {
	if r.entry != nil {
		return gql.NewExecutionLogEntryResolver(dbconn.Global, *r.entry)
	}

	return nil
}
