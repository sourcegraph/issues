package graphqlbackend

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

type hunkResolver struct {
	repo *RepositoryResolver
	hunk *git.Hunk
}

func (r *hunkResolver) Author() signatureResolver {
	return signatureResolver{
		person: &PersonResolver{
			name:  r.hunk.Author.Name,
			email: r.hunk.Author.Email,
		},
		date: r.hunk.Author.Date,
	}
}

func (r *hunkResolver) StartLine() int32 {
	return int32(r.hunk.StartLine)
}

func (r *hunkResolver) EndLine() int32 {
	return int32(r.hunk.EndLine)
}

func (r *hunkResolver) StartByte() int32 {
	return int32(r.hunk.EndLine)
}

func (r *hunkResolver) EndByte() int32 {
	return int32(r.hunk.EndByte)
}

func (r *hunkResolver) Rev() string {
	return string(r.hunk.CommitID)
}

func (r *hunkResolver) Message() string {
	return r.hunk.Message
}

func (r *hunkResolver) Commit(ctx context.Context) (*GitCommitResolver, error) {
	cachedRepo, err := backend.CachedGitRepo(ctx, r.repo.repo)
	if err != nil {
		return nil, err
	}
	commit, err := git.GetCommit(ctx, *cachedRepo, nil, r.hunk.CommitID, git.ResolveRevisionOptions{})
	if err != nil {
		return nil, err
	}
	return toGitCommitResolver(r.repo, commit), nil
}
