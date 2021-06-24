package backend

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

// ResolveRev will return the absolute commit for a commit-ish spec in a repo.
// If no rev is specified, HEAD is used.
// Error cases:
// * Repo does not exist: vcs.RepoNotExistError
// * Commit does not exist: git.RevisionNotFoundError
// * Empty repository: git.RevisionNotFoundError
// * The user does not have permission: errcode.IsNotFound
// * Other unexpected errors.
func (s *Repos) ResolveRev(ctx context.Context, repo *types.Repo, rev string) (commitID api.CommitID, err error) {
	if Mocks.Repos.ResolveRev != nil {
		return Mocks.Repos.ResolveRev(ctx, repo, rev)
	}

	ctx, done := trace(ctx, "Repos", "ResolveRev", map[string]interface{}{"repo": repo.Name, "rev": rev}, &err)
	defer done()

	return git.ResolveRevision(ctx, repo.Name, rev, git.ResolveRevisionOptions{})
}

func (s *Repos) GetCommit(ctx context.Context, repo *types.Repo, commitID api.CommitID) (res *git.Commit, err error) {
	if Mocks.Repos.GetCommit != nil {
		return Mocks.Repos.GetCommit(ctx, repo, commitID)
	}

	ctx, done := trace(ctx, "Repos", "GetCommit", map[string]interface{}{"repo": repo.Name, "commitID": commitID}, &err)
	defer done()

	log15.Debug("svc.local.repos.GetCommit", "repo", repo.Name, "commitID", commitID)

	if !git.IsAbsoluteRevision(string(commitID)) {
		return nil, errors.Errorf("non-absolute CommitID for Repos.GetCommit: %v", commitID)
	}

	return git.GetCommit(ctx, repo.Name, commitID, git.ResolveRevisionOptions{})
}
