package graphqlbackend

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/zoekt"
	zoektquery "github.com/google/zoekt/query"
	"github.com/sourcegraph/sourcegraph/internal/search"
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
	List(ctx context.Context, q zoektquery.Q) (*zoekt.RepoList, error)
}

func (r *repositoryTextSearchIndexResolver) resolve(ctx context.Context) (*zoekt.RepoListEntry, error) {
	r.once.Do(func() {
		repoList, err := r.client.List(ctx, zoektquery.NewRepoSet(string(r.repo.repo.Name)))
		if err != nil {
			r.err = err
			return
		}
		if len(repoList.Repos) > 1 {
			r.err = fmt.Errorf("more than 1 indexed repo found for %q", r.repo.repo.Name)
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
	return int32(r.entry.Stats.Shards + 1)
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
	refByName := func(refName string) *repositoryTextSearchIndexedRef {
		for _, ref := range refs {
			if ref.ref.name == refName {
				return ref
			}
		}

		// If Zoekt reports it has another indexed branch, include that.
		newRef := &repositoryTextSearchIndexedRef{ref: &GitRefResolver{name: refName, repo: r.repo}}
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

	target, err := r.ref.Target().OID(ctx)
	if err != nil {
		return false, err
	}
	return target == r.indexedCommit, nil
}

func (r *repositoryTextSearchIndexedRef) IndexedCommit() *gitObject {
	if r.indexedCommit == "" {
		return nil
	}
	return &gitObject{repo: r.ref.repo, oid: r.indexedCommit, typ: gitObjectTypeCommit}
}
