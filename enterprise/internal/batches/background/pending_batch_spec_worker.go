package background

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gobwas/glob"
	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"
	"github.com/keegancsmith/sqlf"
	"github.com/pkg/errors"

	"github.com/sourcegraph/batch-change-utils/env"
	"github.com/sourcegraph/batch-change-utils/overridable"
	"github.com/sourcegraph/batch-change-utils/yaml"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
	"github.com/sourcegraph/sourcegraph/schema"
)

type pendingBatchSpecHandler struct {
	s *store.Store
}

func (h *pendingBatchSpecHandler) HandlerFunc() dbworker.HandlerFunc {
	return func(ctx context.Context, tx dbworkerstore.Store, record workerutil.Record) error {
		s := h.s.With(tx)
		log15.Info("processing pending batch spec", "record", record, "s", s)

		pbs := record.(pendingBatchSpecRecord).PendingBatchSpec
		spec, err := ParseBatchSpec([]byte(pbs.Spec))
		if err != nil {
			return errors.Wrap(err, "parsing batch spec")
		}

		// We need to resolve the repositories into workspaces, then create
		// BatchExecutionJob instances. We also need to do this as the user, so
		// we'll create a new nested context.
		db := tx.Handle().DB()
		userCtx := actor.WithActor(ctx, &actor.Actor{UID: pbs.CreatorUserID})
		rs := newRepositorySet()
		for _, on := range spec.On {
			if err := resolveRepositoriesOn(userCtx, db, rs, &on); err != nil {
				return errors.Wrapf(err, "resolving on: %+v", on)
			}
			// TODO: handle branch, unsupported repo types, ignored.
		}

		// Now we need to fill in the default branch and commit for each repo.
		for name, head := range rs {
			repo, err := database.Repos(db).GetByName(userCtx, name)
			if err != nil {
				return errors.Wrapf(err, "retrieving repo: %q", name)
			}

			if head.Branch == "" {
				resolver := graphqlbackend.NewRepositoryResolver(db, repo)
				branch, err := resolver.DefaultBranch(userCtx)
				if err != nil {
					return errors.Wrapf(err, "getting default branch for %q", name)
				}
				head.Branch = branch.Name()
			}

			if head.Rev == "" {
				rev, err := git.ResolveRevision(userCtx, name, head.Branch, git.ResolveRevisionOptions{})
				if err != nil {
					return errors.Wrapf(err, "resolving branch %q on repo %q", head.Branch, name)
				}
				head.Rev = string(rev)
			}
		}

		// TODO: implement workspace discovery.

		// Convert spec steps into executor steps.
		steps := make([]executor.DockerStep, len(spec.Steps)+1)
		for i, step := range spec.Steps {
			steps[i] = executor.DockerStep{
				Image:    step.Container,
				Commands: []string{step.Run},
				Env:      []string{},
			}

			// TODO: support outer environments somehow?
			env, err := step.Env.Resolve([]string{})
			if err != nil {
				return errors.Wrapf(err, "resolving environment for step %d", i)
			}
			for k, v := range env {
				steps[i].Env = append(steps[i].Env, k+"="+v)
			}
		}

		// Add a step to dump out the diff.
		//
		// TODO: replace this with a step that can actually create the changeset
		// spec.
		steps[len(spec.Steps)] = executor.DockerStep{
			Image:    "sourcegraph/src-batch-change-volume-workspace",
			Commands: []string{"git diff --cached --no-prefix --binary"},
		}

		// TODO: get the encryption key.
		bstore := store.New(db, nil)

		// Build simple job definitions.
		for repo, head := range rs {
			bej, err := bstore.CreateBatchExecutorJob(ctx, executor.Job{
				Workspace: executor.Workspace{
					RepositoryName: string(repo),
				},
				Commit:      head.Rev,
				DockerSteps: steps,
			})
			if err != nil {
				return errors.Wrapf(err, "creating batch executor job for %q", repo)
			}

			log15.Info("created job", "repo", repo, "head", *head, "bej", *bej)
		}

		return nil
	}
}

func resolveRepositoriesOn(ctx context.Context, db dbutil.DB, rs repositorySet, on *OnQueryOrRepository) error {
	if query := on.RepositoriesMatchingQuery; query != "" {
		return resolveRepositorySearch(ctx, db, rs, query)
	}

	rs.Add(api.RepoName(on.Repository), on.Branch)
	return nil
}

func resolveRepositorySearch(ctx context.Context, db dbutil.DB, rs repositorySet, query string) error {
	impl, err := graphqlbackend.NewSearchImplementer(ctx, db, &graphqlbackend.SearchArgs{
		Version: "V2",
		Query:   query,
	})
	if err != nil {
		return errors.Wrapf(err, "creating search for repository matching query: %q", query)
	}

	resolver, err := impl.Results(ctx)
	if err != nil {
		return errors.Wrapf(err, "creating search result resolver for repository matching query: %q", query)
	}

	for _, match := range resolver.SearchResults {
		rs.Add(match.Key().Repo, "")
	}

	return nil
}

type repositorySet map[api.RepoName]*repositoryHead

type repositoryHead struct {
	Branch string
	Rev    string
}

func newRepositorySet() repositorySet { return repositorySet{} }

func (rs repositorySet) Add(name api.RepoName, branch string) {
	rs[name] = &repositoryHead{
		Branch: branch,
	}
}

type pendingBatchSpecRecord struct {
	*btypes.PendingBatchSpec
}

var _ workerutil.Record = pendingBatchSpecRecord{}

func (r pendingBatchSpecRecord) RecordID() int {
	return int(r.ID)
}

func newPendingBatchSpecWorker(
	ctx context.Context,
	s *store.Store,
	metrics batchChangesMetrics,
) *workerutil.Worker {
	handler := &pendingBatchSpecHandler{s}
	workerStore := newPendingBatchSpecWorkerStore(s)
	options := workerutil.WorkerOptions{
		Name:        "batches_pending_batch_spec_worker",
		NumHandlers: 5,
		Interval:    5 * time.Second,
		Metrics:     metrics.pendingBatchSpecWorkerMetrics,
	}

	return dbworker.NewWorker(ctx, workerStore, handler.HandlerFunc(), options)
}

func newPendingBatchSpecWorkerResetter(s *store.Store, metrics batchChangesMetrics) *dbworker.Resetter {
	workerStore := newPendingBatchSpecWorkerStore(s)
	options := dbworker.ResetterOptions{
		Name:     "batches_pending_batch_spec_worker_resetter",
		Interval: 1 * time.Minute,
		Metrics:  metrics.pendingBatchSpecWorkerResetterMetrics,
	}

	return dbworker.NewResetter(workerStore, options)
}

func newPendingBatchSpecWorkerStore(s *store.Store) dbworkerstore.Store {
	return dbworkerstore.New(s.Handle(), dbworkerstore.Options{
		Name:              "batches_pending_batch_spec_store",
		TableName:         "pending_batch_specs",
		ColumnExpressions: store.PendingBatchSpecColumns,
		Scan: func(rows *sql.Rows, err error) (workerutil.Record, bool, error) {
			pbs, exists, err := store.ScanFirstPendingBatchSpec(rows, err)
			return pendingBatchSpecRecord{pbs}, exists, err
		},
		OrderByExpression: sqlf.Sprintf("pending_batch_specs.updated_at ASC"),
		StalledMaxAge:     60 * time.Second,
		MaxNumResets:      60,
		RetryAfter:        5 * time.Second,
		MaxNumRetries:     60,
	})
}

type BatchSpec struct {
	Name              string                   `json:"name,omitempty" yaml:"name"`
	Description       string                   `json:"description,omitempty" yaml:"description"`
	On                []OnQueryOrRepository    `json:"on,omitempty" yaml:"on"`
	Workspaces        []WorkspaceConfiguration `json:"workspaces,omitempty"  yaml:"workspaces"`
	Steps             []Step                   `json:"steps,omitempty" yaml:"steps"`
	TransformChanges  *TransformChanges        `json:"transformChanges,omitempty" yaml:"transformChanges,omitempty"`
	ImportChangesets  []ImportChangeset        `json:"importChangesets,omitempty" yaml:"importChangesets"`
	ChangesetTemplate *ChangesetTemplate       `json:"changesetTemplate,omitempty" yaml:"changesetTemplate"`
}

type ChangesetTemplate struct {
	Title     string                       `json:"title,omitempty" yaml:"title"`
	Body      string                       `json:"body,omitempty" yaml:"body"`
	Branch    string                       `json:"branch,omitempty" yaml:"branch"`
	Commit    ExpandedGitCommitDescription `json:"commit,omitempty" yaml:"commit"`
	Published overridable.BoolOrString     `json:"published" yaml:"published"`
}

type GitCommitAuthor struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email" yaml:"email"`
}

type ExpandedGitCommitDescription struct {
	Message string           `json:"message,omitempty" yaml:"message"`
	Author  *GitCommitAuthor `json:"author,omitempty" yaml:"author"`
}

type ImportChangeset struct {
	Repository  string        `json:"repository" yaml:"repository"`
	ExternalIDs []interface{} `json:"externalIDs" yaml:"externalIDs"`
}

type WorkspaceConfiguration struct {
	RootAtLocationOf   string `json:"rootAtLocationOf,omitempty" yaml:"rootAtLocationOf"`
	In                 string `json:"in,omitempty" yaml:"in"`
	OnlyFetchWorkspace bool   `json:"onlyFetchWorkspace,omitempty" yaml:"onlyFetchWorkspace"`

	glob glob.Glob
}

type OnQueryOrRepository struct {
	RepositoriesMatchingQuery string `json:"repositoriesMatchingQuery,omitempty" yaml:"repositoriesMatchingQuery"`
	Repository                string `json:"repository,omitempty" yaml:"repository"`
	Branch                    string `json:"branch,omitempty" yaml:"branch"`
}

type Step struct {
	Run       string            `json:"run,omitempty" yaml:"run"`
	Container string            `json:"container,omitempty" yaml:"container"`
	Env       env.Environment   `json:"env,omitempty" yaml:"env"`
	Files     map[string]string `json:"files,omitempty" yaml:"files,omitempty"`
	Outputs   Outputs           `json:"outputs,omitempty" yaml:"outputs,omitempty"`

	If interface{} `json:"if,omitempty" yaml:"if,omitempty"`
}

type Outputs map[string]Output

type Output struct {
	Value  string `json:"value,omitempty" yaml:"value,omitempty"`
	Format string `json:"format,omitempty" yaml:"format,omitempty"`
}

type TransformChanges struct {
	Group []Group `json:"group,omitempty" yaml:"group"`
}

type Group struct {
	Directory  string `json:"directory,omitempty" yaml:"directory"`
	Branch     string `json:"branch,omitempty" yaml:"branch"`
	Repository string `json:"repository,omitempty" yaml:"repository"`
}

func ParseBatchSpec(data []byte) (*BatchSpec, error) {
	var spec BatchSpec
	if err := yaml.UnmarshalValidate(schema.BatchSpecSchemaJSON, data, &spec); err != nil {
		if multiErr, ok := err.(*multierror.Error); ok {
			var newMultiError *multierror.Error

			for _, e := range multiErr.Errors {
				// In case of `name` we try to make the error message more user-friendly.
				if strings.Contains(e.Error(), "name: Does not match pattern") {
					newMultiError = multierror.Append(newMultiError, fmt.Errorf("The batch change name can only contain word characters, dots and dashes. No whitespace or newlines allowed."))
				} else {
					newMultiError = multierror.Append(newMultiError, e)
				}
			}

			return nil, newMultiError.ErrorOrNil()
		}

		return nil, err
	}

	var errs *multierror.Error

	if len(spec.Steps) != 0 && spec.ChangesetTemplate == nil {
		errs = multierror.Append(errs, errors.New("batch spec includes steps but no changesetTemplate"))
	}

	return &spec, errs.ErrorOrNil()
}
