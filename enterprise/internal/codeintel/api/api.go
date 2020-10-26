package api

import (
	"context"
	"errors"

	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
)

// CodeIntelAPI is the main interface into precise code intelligence data.
type CodeIntelAPI interface {
	// FindClosestDumps returns the set of dumps that can most accurately answer code intelligence
	// queries for the given path. If exactPath is true, then only dumps that definitely contain the
	// exact document path are returned. Otherwise, dumps containing any document for which the given
	// path is a prefix are returned. These dump IDs should be subsequently passed to invocations of
	// Definitions, References, and Hover.
	FindClosestDumps(ctx context.Context, repositoryID int, commit, path string, exactPath bool, indexer string) ([]store.Dump, error)

	// Ranges returns definition, reference, and hover data for each range within the given span of lines.
	Ranges(ctx context.Context, file string, startLine, endLine, uploadID int) ([]ResolvedCodeIntelligenceRange, error)

	// Definitions returns the list of source locations that define the symbol at the given position.
	// This may include remote definitions if the remote repository is also indexed.
	Definitions(ctx context.Context, file string, line, character, uploadID int) ([]ResolvedLocation, error)

	// References returns the list of source locations that reference the symbol at the given position.
	// This may include references from other dumps and repositories.
	References(ctx context.Context, repositoryID int, commit string, limit int, cursor Cursor) ([]ResolvedLocation, Cursor, bool, error)

	// Hover returns the hover text and range for the symbol at the given position.
	Hover(ctx context.Context, file string, line, character, uploadID int) (string, bundles.Range, bool, error)

	// Diagnostics returns the diagnostics for documents with the given path prefix.
	Diagnostics(ctx context.Context, prefix string, uploadID, limit, offset int) ([]ResolvedDiagnostic, int, error)
}

type codeIntelAPI struct {
	store               store.Store
	bundleManagerClient bundles.BundleManagerClient
	gitserverClient     gitserverClient
}

type gitserverClient interface {
	CommitGraph(ctx context.Context, store store.Store, repositoryID int, options gitserver.CommitGraphOptions) (map[string][]string, error)
}

var _ CodeIntelAPI = &codeIntelAPI{}

var ErrMissingDump = errors.New("missing dump")

func New(store store.Store, bundleManagerClient bundles.BundleManagerClient, gitserverClient gitserverClient) CodeIntelAPI {
	return &codeIntelAPI{
		store:               store,
		bundleManagerClient: bundleManagerClient,
		gitserverClient:     gitserverClient,
	}
}
