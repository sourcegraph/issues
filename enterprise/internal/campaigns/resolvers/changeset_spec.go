package resolvers

import (
	"context"
	"fmt"
	"sync"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/sourcegraph/go-diff/diff"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

func marshalChangesetSpecRandID(id string) graphql.ID {
	return relay.MarshalID("ChangesetSpec", id)
}

func unmarshalChangesetSpecID(id graphql.ID) (changesetSpecRandID string, err error) {
	err = relay.UnmarshalSpec(id, &changesetSpecRandID)
	return
}

var _ graphqlbackend.ChangesetSpecResolver = &changesetSpecResolver{}

type changesetSpecResolver struct {
	store       *ee.Store
	httpFactory *httpcli.Factory

	changesetSpec *campaigns.ChangesetSpec

	preloadedRepo        *types.Repo
	attemptedPreloadRepo bool

	// Cache repo because it's accessed more than once
	repoOnce sync.Once
	repo     *graphqlbackend.RepositoryResolver
	repoErr  error
	// The context with which we try to load the repository if it's not
	// preloaded. We need an extra field for that, because the
	// ToVisibleChangesetSpec/ToHiddenChangesetSpec methods cannot take a
	// context.Context without graphql-go panic'ing.
	repoCtx context.Context
}

func (r *changesetSpecResolver) ID() graphql.ID {
	// 🚨 SECURITY: This needs to be the RandID! We can't expose the
	// sequential, guessable ID.
	return marshalChangesetSpecRandID(r.changesetSpec.RandID)
}

func (r *changesetSpecResolver) Type() campaigns.ChangesetSpecDescriptionType {
	return r.changesetSpec.Spec.Type()
}

func (r *changesetSpecResolver) computeRepo() (*graphqlbackend.RepositoryResolver, error) {
	r.repoOnce.Do(func() {
		if r.attemptedPreloadRepo {
			if r.preloadedRepo != nil {
				r.repo = graphqlbackend.NewRepositoryResolver(r.preloadedRepo)
			}
		} else {
			if r.repoCtx == nil {
				r.repoErr = fmt.Errorf("no context available to query repository")
				return
			}

			// 🚨 SECURITY: db.Repos.GetByIDs uses the authzFilter under the hood and
			// filters out repositories that the user doesn't have access to.
			// In case we don't find a repository, it might be because it's deleted
			// or because the user doesn't have access.
			repo, err := graphqlbackend.RepositoryByIDInt32(r.repoCtx, r.changesetSpec.RepoID)
			if err != nil && !errcode.IsNotFound(err) {
				r.repoErr = err
				return
			}
			r.repo = repo
		}
	})
	return r.repo, r.repoErr
}

func (r *changesetSpecResolver) Description(ctx context.Context) (graphqlbackend.ChangesetDescription, error) {
	repo, err := r.computeRepo()
	if err != nil {
		return nil, err
	}

	descriptionResolver := &changesetDescriptionResolver{
		desc:         r.changesetSpec.Spec,
		repoResolver: repo,
		diffStat:     r.changesetSpec.DiffStat(),
	}

	return descriptionResolver, nil
}

func (r *changesetSpecResolver) ExpiresAt() *graphqlbackend.DateTime {
	return &graphqlbackend.DateTime{Time: r.changesetSpec.ExpiresAt()}
}

func (r *changesetSpecResolver) repoAccessible() (bool, error) {
	repo, err := r.computeRepo()
	if err != nil {
		// In case we couldn't load the repository because of an error, we
		// return the error
		return false, err
	}

	// If the repository is not nil, it's accessible
	return repo != nil, nil
}

func (r *changesetSpecResolver) ToHiddenChangesetSpec() (graphqlbackend.HiddenChangesetSpecResolver, bool) {
	accessible, err := r.repoAccessible()
	if err != nil {
		return r, true
	}

	if accessible {
		return nil, false
	}

	return r, true
}

func (r *changesetSpecResolver) ToVisibleChangesetSpec() (graphqlbackend.VisibleChangesetSpecResolver, bool) {
	accessible, err := r.repoAccessible()
	if err != nil {
		return r, true
	}

	if !accessible {
		return nil, false
	}

	return r, true
}

var _ graphqlbackend.ChangesetDescription = &changesetDescriptionResolver{}

// changesetDescriptionResolver implements both ChangesetDescription
// interfaces: ExistingChangesetReferenceResolver and
// GitBranchChangesetDescriptionResolver.
type changesetDescriptionResolver struct {
	repoResolver *graphqlbackend.RepositoryResolver
	desc         *campaigns.ChangesetSpecDescription
	diffStat     diff.Stat
}

func (r *changesetDescriptionResolver) ToExistingChangesetReference() (graphqlbackend.ExistingChangesetReferenceResolver, bool) {
	if r.desc.IsImportingExisting() {
		return r, true
	}
	return nil, false
}
func (r *changesetDescriptionResolver) ToGitBranchChangesetDescription() (graphqlbackend.GitBranchChangesetDescriptionResolver, bool) {
	if r.desc.IsBranch() {
		return r, true
	}
	return nil, false
}

func (r *changesetDescriptionResolver) BaseRepository() *graphqlbackend.RepositoryResolver {
	return r.repoResolver
}
func (r *changesetDescriptionResolver) ExternalID() string { return r.desc.ExternalID }
func (r *changesetDescriptionResolver) BaseRef() string    { return git.AbbreviateRef(r.desc.BaseRef) }
func (r *changesetDescriptionResolver) BaseRev() string    { return r.desc.BaseRev }
func (r *changesetDescriptionResolver) HeadRepository() *graphqlbackend.RepositoryResolver {
	return r.repoResolver
}
func (r *changesetDescriptionResolver) HeadRef() string { return git.AbbreviateRef(r.desc.HeadRef) }
func (r *changesetDescriptionResolver) Title() string   { return r.desc.Title }
func (r *changesetDescriptionResolver) Body() string    { return r.desc.Body }
func (r *changesetDescriptionResolver) Published() campaigns.PublishedValue {
	return r.desc.Published
}

func (r *changesetDescriptionResolver) DiffStat() *graphqlbackend.DiffStat {
	return graphqlbackend.NewDiffStat(r.diffStat)
}

func (r *changesetDescriptionResolver) Diff(ctx context.Context) (graphqlbackend.PreviewRepositoryComparisonResolver, error) {
	diff, err := r.desc.Diff()
	if err != nil {
		return nil, err
	}
	return graphqlbackend.NewPreviewRepositoryComparisonResolver(ctx, r.repoResolver, r.desc.BaseRev, diff)
}

func (r *changesetDescriptionResolver) Commits() []graphqlbackend.GitCommitDescriptionResolver {
	var resolvers []graphqlbackend.GitCommitDescriptionResolver
	for _, c := range r.desc.Commits {
		resolvers = append(resolvers, &gitCommitDescriptionResolver{
			message:     c.Message,
			diff:        c.Diff,
			authorName:  c.AuthorName,
			authorEmail: c.AuthorEmail,
		})
	}
	return resolvers
}

var _ graphqlbackend.GitCommitDescriptionResolver = &gitCommitDescriptionResolver{}

type gitCommitDescriptionResolver struct {
	message     string
	diff        string
	authorName  string
	authorEmail string
}

func (r *gitCommitDescriptionResolver) Author() *graphqlbackend.PersonResolver {
	return graphqlbackend.NewPersonResolver(
		r.authorName,
		r.authorEmail,
		// Try to find the corresponding Sourcegraph user.
		true,
	)
}
func (r *gitCommitDescriptionResolver) Message() string { return r.message }
func (r *gitCommitDescriptionResolver) Subject() string {
	return graphqlbackend.GitCommitSubject(r.message)
}
func (r *gitCommitDescriptionResolver) Body() *string {
	body := graphqlbackend.GitCommitBody(r.message)
	if body == "" {
		return nil
	}
	return &body
}
func (r *gitCommitDescriptionResolver) Diff() string { return r.diff }
