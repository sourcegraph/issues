package api

import (
	"context"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
)

// NumAncestors is the number of ancestors to query from gitserver when trying to find the closest
// ancestor we have data for. Setting this value too low (relative to a repository's commit rate)
// will cause requests for an unknown commit return too few results; setting this value too high
// will raise the latency of requests for an unknown commit.
//
// TODO(efritz) - make adjustable
const NumAncestors = 100

// FindClosestDumps returns the set of dumps that can most accurately answer code intelligence
// queries for the given path. If exactPath is true, then only dumps that definitely contain the
// exact document path are returned. Otherwise, dumps containing any document for which the given
// path is a prefix are returned. These dump IDs should be subsequently passed to invocations of
// Definitions, References, and Hover.
func (api *codeIntelAPI) FindClosestDumps(ctx context.Context, repositoryID int, commit, path string, exactPath bool, indexer string) ([]store.Dump, error) {
	candidates, err := api.inferClosestUploads(ctx, repositoryID, commit, path, exactPath, indexer)
	if err != nil {
		return nil, err
	}

	var dumps []store.Dump
	for _, dump := range candidates {
		// TODO(efritz) - ensure there's a valid document path
		// for the other condition. This should probably look like
		// an additional parameter on the following exists query.
		if exactPath {
			exists, err := api.bundleManagerClient.BundleClient(dump.ID).Exists(ctx, strings.TrimPrefix(path, dump.Root))
			if err != nil {
				if err == bundles.ErrNotFound {
					log15.Warn("Bundle does not exist")
					return nil, nil
				}
				return nil, errors.Wrap(err, "bundleManagerClient.BundleClient")
			}
			if !exists {
				continue
			}
		}

		dumps = append(dumps, dump)
	}

	return dumps, nil
}

// inferClosestUploads will return the set of visible uploads for the given commit. If this commit is
// newer than our last refresh of the lsif_nearest_uploads table for this repository, then we will mark
// the repository as dirty and quickly approximate the correct set of visible uploads.
//
// Because updating the entire commit graph is a blocking, expensive, and lock-guarded process, we  want
// to only do that in the background and do something chearp in latency-sensitive paths. To construct an
// approximate result, we query gitserver for a (relatively small) set of ancestors for the given commit,
// correlate that with the upload data we have for those commits, and re-run the visibility algorithm over
// the graph. This will not always produce the full set of visible commits - some responses may not contain
// all results while a subsequent request made after the lsif_nearest_uploads has been updated to include
// this commit will.
//
// TODO(efritz) - show an indication in the GraphQL response and the UI that this repo is refreshing.
func (api *codeIntelAPI) inferClosestUploads(ctx context.Context, repositoryID int, commit, path string, exactPath bool, indexer string) ([]store.Dump, error) {
	commitExists, err := api.store.HasCommit(ctx, repositoryID, commit)
	if err != nil {
		return nil, errors.Wrap(err, "store.HasCommit")
	}
	if commitExists {
		// The parameters exactPath and rootMustEnclosePath align here: if we're looking for dumps
		// that can answer queries for a directory (e.g. diagnostics), we want any dump that happens
		// to intersect the target directory. If we're looking for dumps that can answer queries for
		// a single file, then we need a dump with a root that properly encloses that file.
		dumps, err := api.store.FindClosestDumps(ctx, repositoryID, commit, path, exactPath, indexer)
		if err != nil {
			return nil, errors.Wrap(err, "store.FindClosestDumps")
		}

		return dumps, nil
	}

	repositoryExists, err := api.store.HasRepository(ctx, repositoryID)
	if err != nil {
		return nil, errors.Wrap(err, "store.HasRepository")
	}
	if !repositoryExists {
		// TODO(efritz) - differentiate this error in the GraphQL response/UI
		return nil, nil
	}

	graph, err := api.gitserverClient.CommitGraph(ctx, api.store, repositoryID, gitserver.CommitGraphOptions{
		Commit: commit,
		Limit:  NumAncestors,
	})
	if err != nil {
		return nil, err
	}

	dumps, err := api.store.FindClosestDumpsFromGraphFragment(ctx, repositoryID, commit, path, exactPath, indexer, graph)
	if err != nil {
		return nil, err
	}

	if err := api.store.MarkRepositoryAsDirty(ctx, repositoryID); err != nil {
		return nil, errors.Wrap(err, "store.MarkRepositoryAsDirty")
	}

	return dumps, nil
}
