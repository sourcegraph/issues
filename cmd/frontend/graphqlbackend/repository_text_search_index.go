package graphqlbackend

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/google/zoekt"
	zoektquery "github.com/google/zoekt/query"

	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

func (r *RepositoryResolver) TextSearchIndex() *repositoryTextSearchIndexResolver {
	if !search.Indexed().Enabled() {
		return nil
	}
	return &repositoryTextSearchIndexResolver{
		repo:   r,
		client: search.Indexed().Client,
	}
}

type repositoryTextSearchIndexResolver struct {
	repo   *RepositoryResolver
	client repoLister

	once  sync.Once
	entry *zoekt.RepoListEntry
	err   error
}

type repoLister interface {
	List(ctx context.Context, q zoektquery.Q, opts *zoekt.ListOptions) (*zoekt.RepoList, error)
}

func (r *repositoryTextSearchIndexResolver) resolve(ctx context.Context) (*zoekt.RepoListEntry, error) {
	r.once.Do(func() {
		q := &zoektquery.RepoBranches{Set: map[string][]string{r.repo.Name(): {"HEAD"}}}
		repoList, err := r.client.List(ctx, q, nil)
		if err != nil {
			r.err = err
			return
		}
		if len(repoList.Repos) > 1 {
			r.err = errors.Errorf("more than 1 indexed repo found for %q", r.repo.Name())
			return
		}
		if len(repoList.Repos) == 1 {
			r.entry = repoList.Repos[0]
		}
	})
	return r.entry, r.err
}

func (r *repositoryTextSearchIndexResolver) Repository() *RepositoryResolver { return r.repo }

func (r *repositoryTextSearchIndexResolver) Status(ctx context.Context) (*repositoryTextSearchIndexStatus, error) {
	entry, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}
	return &repositoryTextSearchIndexStatus{entry: *entry}, nil
}

type repositoryTextSearchIndexStatus struct {
	entry zoekt.RepoListEntry
}

func (r *repositoryTextSearchIndexStatus) UpdatedAt() DateTime {
	return DateTime{Time: r.entry.IndexMetadata.IndexTime}
}

func (r *repositoryTextSearchIndexStatus) ContentByteSize() int32 {
	return int32(r.entry.Stats.ContentBytes)
}

func (r *repositoryTextSearchIndexStatus) ContentFilesCount() int32 {
	return int32(r.entry.Stats.Documents)
}

func (r *repositoryTextSearchIndexStatus) IndexByteSize() int32 {
	return int32(r.entry.Stats.IndexBytes)
}

func (r *repositoryTextSearchIndexStatus) IndexShardsCount() int32 {
	return int32(r.entry.Stats.Shards)
}

func (r *repositoryTextSearchIndexStatus) NewLinesCount() int32 {
	return int32(r.entry.Stats.NewLinesCount)
}

func (r *repositoryTextSearchIndexStatus) DefaultBranchNewLinesCount() int32 {
	return int32(r.entry.Stats.DefaultBranchNewLinesCount)
}

func (r *repositoryTextSearchIndexStatus) OtherBranchesNewLinesCount() int32 {
	return int32(r.entry.Stats.OtherBranchesNewLinesCount)
}

func (r *repositoryTextSearchIndexResolver) Refs(ctx context.Context) ([]*repositoryTextSearchIndexedRef, error) {
	// We assume that the default branch for enabled repositories is always configured to be indexed.
	//
	// TODO(sqs): support configuring which branches should be indexed (add'l branches, not default branch, etc.).
	defaultBranchRef, err := r.repo.DefaultBranch(ctx)
	if err != nil {
		return nil, err
	}
	if defaultBranchRef == nil {
		return []*repositoryTextSearchIndexedRef{}, nil
	}
	refNames := []string{defaultBranchRef.name}

	refs := make([]*repositoryTextSearchIndexedRef, len(refNames))
	for i, refName := range refNames {
		refs[i] = &repositoryTextSearchIndexedRef{ref: &GitRefResolver{name: refName, repo: r.repo}}
	}
	refByName := func(name string) *repositoryTextSearchIndexedRef {
		possibleRefNames := []string{"refs/heads/" + name, "refs/tags/" + name}
		for _, ref := range possibleRefNames {
			if _, err := git.ResolveRevision(ctx, r.repo.RepoName(), ref, git.ResolveRevisionOptions{NoEnsureRevision: true}); err == nil {
				name = ref
				break
			}
		}
		for _, ref := range refs {
			if ref.ref.name == name {
				return ref
			}
		}

		// If Zoekt reports it has another indexed branch, include that.
		newRef := &repositoryTextSearchIndexedRef{ref: &GitRefResolver{name: name, repo: r.repo}}
		refs = append(refs, newRef)
		return newRef
	}

	entry, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	if entry != nil {
		for _, branch := range entry.Repository.Branches {
			name := branch.Name
			if branch.Name == "HEAD" {
				name = defaultBranchRef.name
			}
			ref := refByName(name)
			ref.indexedCommit = GitObjectID(branch.Version)
		}
	}
	return refs, nil
}

type repositoryTextSearchIndexedRef struct {
	ref           *GitRefResolver
	indexedCommit GitObjectID
}

func (r *repositoryTextSearchIndexedRef) Ref() *GitRefResolver { return r.ref }
func (r *repositoryTextSearchIndexedRef) Indexed() bool        { return r.indexedCommit != "" }

func (r *repositoryTextSearchIndexedRef) Current(ctx context.Context) (bool, error) {
	if r.indexedCommit == "" {
		return false, nil
	}

	commit, err := r.ref.Target().Commit(ctx)
	if err != nil {
		return false, err
	}
	return commit.oid == r.indexedCommit, nil
}

func (r *repositoryTextSearchIndexedRef) IndexedCommit() *gitObject {
	if r.indexedCommit == "" {
		return nil
	}
	return &gitObject{repo: r.ref.repo, oid: r.indexedCommit, typ: gitObjectTypeCommit}
}
