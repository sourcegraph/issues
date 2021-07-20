package graphql

import (
	"fmt"
	"strings"

	gql "github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	store "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

// indexStepsResolver resolves the steps of an index record.
//
// Index jobs are broken into three parts:
//   - pre-index steps; all but the last docker step
//   - index step; the last docker step
//   - upload step; the only src-cli step
//
// The setup and teardown steps match the executor setup and teardown.
type indexStepsResolver struct {
	index store.Index
}

var _ gql.IndexStepsResolver = &indexStepsResolver{}

func (r *indexStepsResolver) Setup() []gql.ExecutionLogEntryResolver {
	return r.executionLogEntryResolversWithPrefix("setup.")
}

func (r *indexStepsResolver) PreIndex() []gql.PreIndexStepResolver {
	var resolvers []gql.PreIndexStepResolver
	for i, step := range r.index.DockerSteps {
		if entry, ok := r.findExecutionLogEntry(fmt.Sprintf("step.docker.%d", i)); ok {
			resolvers = append(resolvers, &preIndexStepResolver{step: step, entry: &entry})
		} else {
			resolvers = append(resolvers, &preIndexStepResolver{step: step, entry: nil})
		}
	}

	return resolvers
}

func (r *indexStepsResolver) Index() gql.IndexStepResolver {
	if entry, ok := r.findExecutionLogEntry(fmt.Sprintf("step.docker.%d", len(r.index.DockerSteps))); ok {
		return &indexStepResolver{index: r.index, entry: &entry}
	}

	return &indexStepResolver{index: r.index, entry: nil}
}

func (r *indexStepsResolver) Upload() gql.ExecutionLogEntryResolver {
	if entry, ok := r.findExecutionLogEntry("step.src.0"); ok {
		return gql.NewExecutionLogEntryResolver(dbconn.Global, entry)
	}

	return nil
}

func (r *indexStepsResolver) Teardown() []gql.ExecutionLogEntryResolver {
	return r.executionLogEntryResolversWithPrefix("teardown.")
}

func (r *indexStepsResolver) findExecutionLogEntry(key string) (workerutil.ExecutionLogEntry, bool) {
	for _, entry := range r.index.ExecutionLogs {
		if entry.Key == key {
			return entry, true
		}
	}

	return workerutil.ExecutionLogEntry{}, false
}

func (r *indexStepsResolver) executionLogEntryResolversWithPrefix(prefix string) []gql.ExecutionLogEntryResolver {
	var resolvers []gql.ExecutionLogEntryResolver
	for _, entry := range r.index.ExecutionLogs {
		if !strings.HasPrefix(entry.Key, prefix) {
			continue
		}
		r := gql.NewExecutionLogEntryResolver(dbconn.Global, entry)
		resolvers = append(resolvers, r)
	}

	return resolvers
}
