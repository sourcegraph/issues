package resolvers

import (
	"context"
	"encoding/json"

	codeintelapi "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/api"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
)

// AdjustedLocation is similar to a codeintelapi.ResolvedLocation, but with fields denoting
// the commit and range adjusted for the target commit (when the requested commit is not indexed).
type AdjustedLocation struct {
	Dump           store.Dump
	Path           string
	AdjustedCommit string
	AdjustedRange  bundles.Range
}

// AdjustedDiagnostic is similar to a codeintelapi.ResolvedDiagnostic, but with fields denoting
// the commit and range adjusted for the target commit (when the requested commit is not indexed).
type AdjustedDiagnostic struct {
	bundles.Diagnostic
	Dump           store.Dump
	AdjustedCommit string
	AdjustedRange  bundles.Range
}

// AdjustedCodeIntelligenceRange is similar to a codeintelapi.CodeIntelligenceRange,
// but with adjusted definition and reference locations.
type AdjustedCodeIntelligenceRange struct {
	Range       bundles.Range
	Definitions []AdjustedLocation
	References  []AdjustedLocation
	HoverText   string
}

// QueryResolver is the main interface to bundle-related operations exposed to the GraphQL API. This
// resolver consolidates the logic for bundle operations and is not itself concerned with GraphQL/API
// specifics (auth, validation, marshaling, etc.). This resolver is wrapped by a symmetrics resolver
// in this package's graphql subpackage, which is exposed directly by the API.
type QueryResolver interface {
	Ranges(ctx context.Context, startLine, endLine int) ([]AdjustedCodeIntelligenceRange, error)
	Definitions(ctx context.Context, line, character int) ([]AdjustedLocation, error)
	References(ctx context.Context, line, character, limit int, rawCursor string) ([]AdjustedLocation, string, error)
	Hover(ctx context.Context, line, character int) (string, bundles.Range, bool, error)
	Diagnostics(ctx context.Context, limit int) ([]AdjustedDiagnostic, int, error)
}

type queryResolver struct {
	store               store.Store
	bundleManagerClient bundles.BundleManagerClient
	codeIntelAPI        codeintelapi.CodeIntelAPI
	positionAdjuster    PositionAdjuster
	repositoryID        int
	commit              string
	path                string
	uploads             []store.Dump
}

// NewQueryResolver create a new query resolver with the given services. The methods of this
// struct return queries for the given repository, commit, and path, and will query only the
// bundles associated with the given dump objects.
func NewQueryResolver(
	store store.Store,
	bundleManagerClient bundles.BundleManagerClient,
	codeIntelAPI codeintelapi.CodeIntelAPI,
	positionAdjuster PositionAdjuster,
	repositoryID int,
	commit string,
	path string,
	uploads []store.Dump,
) QueryResolver {
	return &queryResolver{
		store:               store,
		bundleManagerClient: bundleManagerClient,
		codeIntelAPI:        codeIntelAPI,
		positionAdjuster:    positionAdjuster,
		repositoryID:        repositoryID,
		commit:              commit,
		path:                path,
		uploads:             uploads,
	}
}

// Ranges returns code intelligence for the ranges that fall within the given range of lines. These
// results are partial and do not include references outside the current file, or any location that
// requires cross-linking of bundles (cross-repo or cross-root).
func (r *queryResolver) Ranges(ctx context.Context, startLine, endLine int) ([]AdjustedCodeIntelligenceRange, error) {
	var adjustedRanges []AdjustedCodeIntelligenceRange
	for i := range r.uploads {
		adjustedPath, ok, err := r.positionAdjuster.AdjustPath(ctx, r.uploads[i].Commit, r.path, false)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		// TODO(efritz) - determine how to do best-effort line adjustments for this case
		ranges, err := r.codeIntelAPI.Ranges(ctx, adjustedPath, startLine, endLine, r.uploads[i].ID)
		if err != nil {
			return nil, err
		}

		for _, rn := range ranges {
			adjustedDefinitions, err := r.adjustLocations(ctx, rn.Definitions)
			if err != nil {
				return nil, err
			}

			adjustedReferences, err := r.adjustLocations(ctx, rn.References)
			if err != nil {
				return nil, err
			}

			_, adjustedRange, err := r.adjustRange(ctx, r.uploads[i].RepositoryID, r.uploads[i].Commit, adjustedPath, rn.Range)
			if err != nil {
				return nil, err
			}

			adjustedRanges = append(adjustedRanges, AdjustedCodeIntelligenceRange{
				Range:       adjustedRange,
				Definitions: adjustedDefinitions,
				References:  adjustedReferences,
				HoverText:   rn.HoverText,
			})
		}
	}

	return adjustedRanges, nil
}

// Definitions returns the list of source locations that define the symbol at the given position.
// This may include remote definitions if the remote repository is also indexed. If there are multiple
// bundles associated with this resolver, the definitions from the first bundle with any results will
// be returned.
func (r *queryResolver) Definitions(ctx context.Context, line, character int) ([]AdjustedLocation, error) {
	position := bundles.Position{Line: line, Character: character}

	for i := range r.uploads {
		adjustedPath, adjustedPosition, ok, err := r.positionAdjuster.AdjustPosition(ctx, r.uploads[i].Commit, r.path, position, false)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		locations, err := r.codeIntelAPI.Definitions(ctx, adjustedPath, adjustedPosition.Line, adjustedPosition.Character, r.uploads[i].ID)
		if err != nil {
			return nil, err
		}
		if len(locations) == 0 {
			continue
		}

		return r.adjustLocations(ctx, locations)
	}

	return nil, nil
}

// References returns the list of source locations that reference the symbol at the given position.
// This may include references from other dumps and repositories. If there are multiple bundles
// associated with this resolver, results from all bundles will be concatenated and returned.
func (r *queryResolver) References(ctx context.Context, line, character, limit int, rawCursor string) ([]AdjustedLocation, string, error) {
	position := bundles.Position{Line: line, Character: character}

	// Decode a map of upload ids to the next url that serves
	// the new page of results. This may not include an entry
	// for every upload if their result sets have already been
	// exhausted.
	cursors, err := readCursor(rawCursor)
	if err != nil {
		return nil, "", err
	}

	// We need to maintain a symmetric map for the next page
	// of results that we can encode into the endCursor of
	// this request.
	newCursors := map[int]string{}

	var allLocations []codeintelapi.ResolvedLocation
	for i := range r.uploads {
		rawCursor := ""
		if cursor, ok := cursors[r.uploads[i].ID]; ok {
			rawCursor = cursor
		} else if len(cursors) != 0 {
			// Result set is exhausted or newer than the first page
			// of results. Skip anything from this upload as it will
			// have duplicate results, or it will be out of order.
			continue
		}

		adjustedPath, adjustedPosition, ok, err := r.positionAdjuster.AdjustPosition(ctx, r.uploads[i].Commit, r.path, position, false)
		if err != nil {
			return nil, "", err
		}
		if !ok {
			continue
		}

		cursor, err := codeintelapi.DecodeOrCreateCursor(adjustedPath, adjustedPosition.Line, adjustedPosition.Character, r.uploads[i].ID, rawCursor, r.store, r.bundleManagerClient)
		if err != nil {
			return nil, "", err
		}

		locations, newCursor, hasNewCursor, err := r.codeIntelAPI.References(ctx, r.repositoryID, r.commit, limit, cursor)
		if err != nil {
			return nil, "", err
		}

		allLocations = append(allLocations, locations...)
		if hasNewCursor {
			newCursors[r.uploads[i].ID] = codeintelapi.EncodeCursor(newCursor)
		}
	}

	endCursor, err := makeCursor(newCursors)
	if err != nil {
		return nil, "", err
	}

	adjustedLocations, err := r.adjustLocations(ctx, allLocations)
	if err != nil {
		return nil, "", err
	}

	return adjustedLocations, endCursor, nil
}

// Hover returns the hover text and range for the symbol at the given position. If there are
// multiple bundles associated with this resolver, the hover text and range from the first
// bundle with any results will be returned.
func (r *queryResolver) Hover(ctx context.Context, line, character int) (string, bundles.Range, bool, error) {
	position := bundles.Position{Line: line, Character: character}

	for i := range r.uploads {
		adjustedPath, adjustedPosition, ok, err := r.positionAdjuster.AdjustPosition(ctx, r.uploads[i].Commit, r.path, position, false)
		if err != nil {
			return "", bundles.Range{}, false, err
		}
		if !ok {
			continue
		}

		text, rn, exists, err := r.codeIntelAPI.Hover(ctx, adjustedPath, adjustedPosition.Line, adjustedPosition.Character, r.uploads[i].ID)
		if err != nil {
			return "", bundles.Range{}, false, err
		}
		if !exists || text == "" {
			continue
		}

		if _, adjustedRange, ok, err := r.positionAdjuster.AdjustRange(ctx, r.uploads[i].Commit, r.path, rn, true); err != nil {
			return "", bundles.Range{}, false, err
		} else if ok {
			return text, adjustedRange, true, nil
		}

		// Failed to adjust range. This _might_ happen in cases where the LSIF range
		// spans multiple lines which intersect a diff; the hover position on an earlier
		// line may not be edited, but the ending line of the expression may have been
		// edited or removed. This is rare and unfortunate, and we'll skip the result
		// in this case because we have low confidence that it will be rendered correctly.
		continue
	}

	return "", bundles.Range{}, false, nil
}

// Diagnostics returns the diagnostics for documents with the given path prefix. If there are
// multiple bundles associated with this resolver, results from all bundles will be concatenated
// and returned.
func (r *queryResolver) Diagnostics(ctx context.Context, limit int) ([]AdjustedDiagnostic, int, error) {
	totalCount := 0
	var allDiagnostics []codeintelapi.ResolvedDiagnostic
	for i := range r.uploads {
		adjustedPath, ok, err := r.positionAdjuster.AdjustPath(ctx, r.uploads[i].Commit, r.path, false)
		if err != nil {
			return nil, 0, err
		}
		if !ok {
			continue
		}

		l := limit - len(allDiagnostics)
		if l < 0 {
			l = 0
		}

		diagnostics, count, err := r.codeIntelAPI.Diagnostics(ctx, adjustedPath, r.uploads[i].ID, l, 0)
		if err != nil {
			return nil, 0, err
		}

		totalCount += count
		allDiagnostics = append(allDiagnostics, diagnostics...)
	}

	adjustedDiagnostics := make([]AdjustedDiagnostic, 0, len(allDiagnostics))
	for i := range allDiagnostics {
		clientRange := bundles.Range{
			Start: bundles.Position{Line: allDiagnostics[i].Diagnostic.StartLine, Character: allDiagnostics[i].Diagnostic.StartCharacter},
			End:   bundles.Position{Line: allDiagnostics[i].Diagnostic.EndLine, Character: allDiagnostics[i].Diagnostic.EndCharacter},
		}

		adjustedCommit, adjustedRange, err := r.adjustRange(ctx, allDiagnostics[i].Dump.RepositoryID, allDiagnostics[i].Dump.Commit, allDiagnostics[i].Diagnostic.Path, clientRange)
		if err != nil {
			return nil, 0, err
		}

		adjustedDiagnostics = append(adjustedDiagnostics, AdjustedDiagnostic{
			Diagnostic:     allDiagnostics[i].Diagnostic,
			Dump:           allDiagnostics[i].Dump,
			AdjustedCommit: adjustedCommit,
			AdjustedRange:  adjustedRange,
		})
	}

	return adjustedDiagnostics, totalCount, nil
}

// adjustLocations translates a list of resolved locations (relative to the indexed commit) into a list of
// equivalent locations in the requested commit.
func (r *queryResolver) adjustLocations(ctx context.Context, locations []codeintelapi.ResolvedLocation) ([]AdjustedLocation, error) {
	adjustedLocations := make([]AdjustedLocation, 0, len(locations))
	for i := range locations {
		adjustedCommit, adjustedRange, err := r.adjustRange(ctx, locations[i].Dump.RepositoryID, locations[i].Dump.Commit, locations[i].Path, locations[i].Range)
		if err != nil {
			return nil, err
		}

		adjustedLocations = append(adjustedLocations, AdjustedLocation{
			Dump:           locations[i].Dump,
			Path:           locations[i].Path,
			AdjustedCommit: adjustedCommit,
			AdjustedRange:  adjustedRange,
		})
	}

	return adjustedLocations, nil
}

// adjustRange translates a range (relative to the indexed commit) into an equivalent range in the requested commit.
func (r *queryResolver) adjustRange(ctx context.Context, repositoryID int, commit, path string, rx bundles.Range) (string, bundles.Range, error) {
	if repositoryID != r.repositoryID {
		// No diffs exist for translation between repos
		return commit, rx, nil
	}

	if _, adjustedRange, ok, err := r.positionAdjuster.AdjustRange(ctx, commit, path, rx, true); err != nil {
		return "", bundles.Range{}, err
	} else if ok {
		return r.commit, adjustedRange, nil
	}

	return commit, rx, nil
}

// readCursor decodes a cursor into a map from upload ids to URLs that serves the next page of results.
func readCursor(after string) (map[int]string, error) {
	if after == "" {
		return nil, nil
	}

	var cursors map[int]string
	if err := json.Unmarshal([]byte(after), &cursors); err != nil {
		return nil, err
	}
	return cursors, nil
}

// makeCursor encodes a map from upload ids to URLs that serves the next page of results into a single string
// that can be sent back for use in cursor pagination.
func makeCursor(cursors map[int]string) (string, error) {
	if len(cursors) == 0 {
		return "", nil
	}

	encoded, err := json.Marshal(cursors)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
