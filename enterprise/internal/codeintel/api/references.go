package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/inconshreveable/log15"
	pkgerrors "github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bloomfilter"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
)

// RemoteDumpLimit is the limit for fetching batches of remote dumps.
const RemoteDumpLimit = 20

// ErrIllegalLimit occurs when a zero-length page of references is requested
var ErrIllegalLimit = errors.New("limit must be positive")

// References returns the list of source locations that reference the symbol at the given position.
// This may include references from other dumps and repositories.
func (api *codeIntelAPI) References(ctx context.Context, repositoryID int, commit string, limit int, cursor Cursor) ([]ResolvedLocation, Cursor, bool, error) {
	if limit <= 0 {
		return nil, Cursor{}, false, ErrIllegalLimit
	}

	rpr := &ReferencePageResolver{
		store:               api.store,
		bundleManagerClient: api.bundleManagerClient,
		repositoryID:        repositoryID,
		commit:              commit,
		remoteDumpLimit:     RemoteDumpLimit,
		limit:               limit,
	}

	return rpr.resolvePage(ctx, cursor)
}

type ReferencePageResolver struct {
	store               store.Store
	bundleManagerClient bundles.BundleManagerClient
	repositoryID        int
	commit              string
	remoteDumpLimit     int
	limit               int
}

func (s *ReferencePageResolver) resolvePage(ctx context.Context, cursor Cursor) ([]ResolvedLocation, Cursor, bool, error) {
	var allLocations []ResolvedLocation

	for {
		locations, newCursor, hasNewCursor, err := s.dispatchCursorHandler(ctx, cursor)
		if err != nil {
			return nil, Cursor{}, false, err
		}

		s.limit -= len(locations)
		allLocations = append(allLocations, locations...)

		if !hasNewCursor || s.limit <= 0 {
			return allLocations, newCursor, hasNewCursor, err
		}

		cursor = newCursor
	}
}

func (s *ReferencePageResolver) dispatchCursorHandler(ctx context.Context, cursor Cursor) ([]ResolvedLocation, Cursor, bool, error) {
	fns := map[string]func(context.Context, Cursor) ([]ResolvedLocation, Cursor, bool, error){
		"same-dump":           s.handleSameDumpCursor,
		"same-dump-monikers":  s.handleSameDumpMonikersCursor,
		"definition-monikers": s.handleDefinitionMonikersCursor,
		"same-repo":           s.handleSameRepoCursor,
		"remote-repo":         s.handleRemoteRepoCursor,
	}

	fn, exists := fns[cursor.Phase]
	if !exists {
		return nil, Cursor{}, false, fmt.Errorf("unknown cursor phase %s", cursor.Phase)
	}

	locations, newCursor, hasNewCursor, err := fn(ctx, cursor)
	if err != nil {
		return nil, Cursor{}, false, pkgerrors.Wrap(err, cursor.Phase)
	}
	return locations, newCursor, hasNewCursor, nil
}

func (s *ReferencePageResolver) handleSameDumpCursor(ctx context.Context, cursor Cursor) ([]ResolvedLocation, Cursor, bool, error) {
	dump, exists, err := s.store.GetDumpByID(ctx, cursor.DumpID)
	if err != nil {
		return nil, Cursor{}, false, pkgerrors.Wrap(err, "store.GetDumpByID")
	}
	if !exists {
		return nil, Cursor{}, false, ErrMissingDump
	}
	bundleClient := s.bundleManagerClient.BundleClient(dump.ID)

	locations, err := bundleClient.References(ctx, cursor.Path, cursor.Line, cursor.Character)
	if err != nil {
		if err == bundles.ErrNotFound {
			log15.Warn("Bundle does not exist")
			return nil, Cursor{}, false, nil
		}
		return nil, Cursor{}, false, pkgerrors.Wrap(err, "bundleClient.References")
	}

	resolvedLocations := resolveLocationsWithDump(dump, sliceLocations(locations, cursor.SkipResults, cursor.SkipResults+s.limit))

	if newOffset := cursor.SkipResults + s.limit; newOffset <= len(locations) {
		newCursor := Cursor{
			Phase:       cursor.Phase,
			DumpID:      cursor.DumpID,
			Path:        cursor.Path,
			Line:        cursor.Line,
			Character:   cursor.Character,
			Monikers:    cursor.Monikers,
			SkipResults: newOffset,
		}
		return resolvedLocations, newCursor, true, nil
	}

	newCursor := Cursor{
		Phase:       "same-dump-monikers",
		DumpID:      cursor.DumpID,
		Path:        cursor.Path,
		Line:        cursor.Line,
		Character:   cursor.Character,
		Monikers:    cursor.Monikers,
		SkipResults: 0,
	}
	return resolvedLocations, newCursor, true, nil
}

func (s *ReferencePageResolver) handleSameDumpMonikersCursor(ctx context.Context, cursor Cursor) ([]ResolvedLocation, Cursor, bool, error) {
	dump, exists, err := s.store.GetDumpByID(ctx, cursor.DumpID)
	if err != nil {
		return nil, Cursor{}, false, pkgerrors.Wrap(err, "store.GetDumpByID")
	}
	if !exists {
		return nil, Cursor{}, false, ErrMissingDump
	}
	bundleClient := s.bundleManagerClient.BundleClient(dump.ID)

	// Get the references that we've seen from the graph-encoded portion of the bundle. We
	// need to know what we've returned previously so that we can filter out duplicate locations
	// that are also encoded as monikers.
	previousLocations, err := bundleClient.References(ctx, cursor.Path, cursor.Line, cursor.Character)
	if err != nil {
		if err == bundles.ErrNotFound {
			log15.Warn("Bundle does not exist")
			return nil, Cursor{}, false, nil
		}
		return nil, Cursor{}, false, pkgerrors.Wrap(err, "bundleClient.References")
	}

	hashes := map[string]struct{}{}
	for _, location := range previousLocations {
		hashes[hashLocation(location)] = struct{}{}
	}

	var totalCount int
	var locations []bundles.Location

	// Search the references table of the current dump. This search is necessary because
	// we want a 'Find References' operation on a reference to also return references to
	// the governing definition, and those may not be fully linked in the LSIF data. This
	// method returns a cursor if there are reference rows remaining for a subsequent page.
	for _, moniker := range cursor.Monikers {
		results, count, err := bundleClient.MonikerResults(ctx, "reference", moniker.Scheme, moniker.Identifier, cursor.SkipResults, s.limit)
		if err != nil {
			if err == bundles.ErrNotFound {
				log15.Warn("Bundle does not exist")
				return nil, Cursor{}, false, nil
			}
			return nil, Cursor{}, false, pkgerrors.Wrap(err, "bundleClient.MonikerResults")
		}

		if count > 0 {
			for _, location := range results {
				if _, ok := hashes[hashLocation(location)]; !ok {
					locations = append(locations, location)
				}
			}

			totalCount = count
			break
		}
	}

	resolvedLocations := resolveLocationsWithDump(dump, locations)

	if newOffset := cursor.SkipResults + s.limit; newOffset <= totalCount {
		newCursor := Cursor{
			Phase:       cursor.Phase,
			DumpID:      cursor.DumpID,
			Path:        cursor.Path,
			Line:        cursor.Line,
			Character:   cursor.Character,
			Monikers:    cursor.Monikers,
			SkipResults: newOffset,
		}
		return resolvedLocations, newCursor, true, nil
	}

	newCursor := Cursor{
		DumpID:      cursor.DumpID,
		Phase:       "definition-monikers",
		Path:        cursor.Path,
		Monikers:    cursor.Monikers,
		SkipResults: 0,
	}
	return resolvedLocations, newCursor, true, nil
}

func (s *ReferencePageResolver) handleDefinitionMonikersCursor(ctx context.Context, cursor Cursor) ([]ResolvedLocation, Cursor, bool, error) {
	var hasNextPhaseCursor = false
	var nextPhaseCursor Cursor
	for _, moniker := range cursor.Monikers {
		if moniker.PackageInformationID == "" {
			continue
		}

		packageInformation, err := s.bundleManagerClient.BundleClient(cursor.DumpID).PackageInformation(ctx, cursor.Path, moniker.PackageInformationID)
		if err != nil {
			if err == bundles.ErrNotFound {
				log15.Warn("Bundle does not exist")
				return nil, Cursor{}, false, nil
			}
			return nil, Cursor{}, false, pkgerrors.Wrap(err, "bundleManagerClient.PackageInformation")
		}

		hasNextPhaseCursor = true
		nextPhaseCursor = Cursor{
			DumpID:                 cursor.DumpID,
			Phase:                  "same-repo",
			Scheme:                 moniker.Scheme,
			Identifier:             moniker.Identifier,
			Name:                   packageInformation.Name,
			Version:                packageInformation.Version,
			DumpIDs:                nil,
			TotalDumpsWhenBatching: 0,
			SkipDumpsWhenBatching:  0,
			SkipDumpsInBatch:       0,
			SkipResultsInDump:      0,
		}
		break
	}

	for _, moniker := range cursor.Monikers {
		if moniker.Kind != "import" {
			continue
		}

		locations, count, err := lookupMoniker(s.store, s.bundleManagerClient, cursor.DumpID, cursor.Path, "reference", moniker, cursor.SkipResults, s.limit)
		if err != nil {
			return nil, Cursor{}, false, err
		}
		if len(locations) == 0 {
			continue
		}

		if newOffset := cursor.SkipResults + len(locations); newOffset < count {
			newCursor := Cursor{
				Phase:       cursor.Phase,
				DumpID:      cursor.DumpID,
				Path:        cursor.Path,
				Monikers:    cursor.Monikers,
				SkipResults: newOffset,
			}
			return locations, newCursor, true, nil
		}

		return locations, nextPhaseCursor, hasNextPhaseCursor, nil
	}

	return nil, nextPhaseCursor, hasNextPhaseCursor, nil

}

func (s *ReferencePageResolver) handleSameRepoCursor(ctx context.Context, cursor Cursor) ([]ResolvedLocation, Cursor, bool, error) {
	locations, newCursor, hasNewCursor, err := s.resolveLocationsViaReferencePager(ctx, cursor, func(ctx context.Context) (int, store.ReferencePager, error) {
		totalCount, pager, err := s.store.SameRepoPager(ctx, s.repositoryID, s.commit, cursor.Scheme, cursor.Name, cursor.Version, s.remoteDumpLimit)
		if err != nil {
			return 0, nil, pkgerrors.Wrap(err, "store.SameRepoPager")
		}
		return totalCount, pager, nil
	})
	if err != nil || hasNewCursor {
		return locations, newCursor, hasNewCursor, err
	}

	newCursor = Cursor{
		DumpID:                 cursor.DumpID,
		Phase:                  "remote-repo",
		Scheme:                 cursor.Scheme,
		Identifier:             cursor.Identifier,
		Name:                   cursor.Name,
		Version:                cursor.Version,
		DumpIDs:                nil,
		TotalDumpsWhenBatching: 0,
		SkipDumpsWhenBatching:  0,
		SkipDumpsInBatch:       0,
		SkipResultsInDump:      0,
	}
	return locations, newCursor, true, nil
}

func (s *ReferencePageResolver) handleRemoteRepoCursor(ctx context.Context, cursor Cursor) ([]ResolvedLocation, Cursor, bool, error) {
	return s.resolveLocationsViaReferencePager(ctx, cursor, func(ctx context.Context) (int, store.ReferencePager, error) {
		totalCount, pager, err := s.store.PackageReferencePager(ctx, cursor.Scheme, cursor.Name, cursor.Version, s.repositoryID, s.remoteDumpLimit)
		if err != nil {
			return 0, nil, pkgerrors.Wrap(err, "store.PackageReferencePager")
		}
		return totalCount, pager, nil
	})
}

func (s *ReferencePageResolver) resolveLocationsViaReferencePager(ctx context.Context, cursor Cursor, createPager func(context.Context) (int, store.ReferencePager, error)) ([]ResolvedLocation, Cursor, bool, error) {
	dumpID := cursor.DumpID
	scheme := cursor.Scheme
	identifier := cursor.Identifier
	limit := s.limit

	if len(cursor.DumpIDs) == 0 {
		totalCount, pager, err := createPager(ctx)
		if err != nil {
			return nil, Cursor{}, false, err
		}

		identifier := cursor.Identifier
		offset := cursor.SkipDumpsWhenBatching
		limit := s.remoteDumpLimit
		newOffset := offset

		var packageReferences []types.PackageReference
		for len(packageReferences) < limit && newOffset < totalCount {
			page, err := pager.PageFromOffset(ctx, newOffset)
			if err != nil {
				return nil, Cursor{}, false, pager.Done(err)
			}

			if len(page) == 0 {
				// Shouldn't happen, but just in case of a bug we
				// don't want this to throw up into an infinite loop.
				break
			}

			filtered, scanned := applyBloomFilter(page, identifier, limit-len(packageReferences))
			packageReferences = append(packageReferences, filtered...)
			newOffset += scanned
		}

		var dumpIDs []int
		for _, ref := range packageReferences {
			dumpIDs = append(dumpIDs, ref.DumpID)
		}

		cursor.DumpIDs = dumpIDs
		cursor.SkipDumpsWhenBatching = newOffset
		cursor.TotalDumpsWhenBatching = totalCount

		if err := pager.Done(nil); err != nil {
			return nil, Cursor{}, false, err
		}
	}

	for i, batchDumpID := range cursor.DumpIDs {
		// Skip the remote reference that show up for ourselves - we've already gathered
		// these in the previous step of the references query.
		if i < cursor.SkipDumpsInBatch || batchDumpID == dumpID {
			continue
		}

		dump, exists, err := s.store.GetDumpByID(ctx, batchDumpID)
		if err != nil {
			return nil, Cursor{}, false, pkgerrors.Wrap(err, "store.GetDumpByID")
		}
		if !exists {
			continue
		}
		bundleClient := s.bundleManagerClient.BundleClient(batchDumpID)

		results, count, err := bundleClient.MonikerResults(ctx, "reference", scheme, identifier, cursor.SkipResultsInDump, limit)
		if err != nil {
			if err == bundles.ErrNotFound {
				log15.Warn("Bundle does not exist")
				return nil, Cursor{}, false, nil
			}
			return nil, Cursor{}, false, pkgerrors.Wrap(err, "bundleClient.MonikerResults")
		}
		if len(results) == 0 {
			continue
		}
		resolvedLocations := resolveLocationsWithDump(dump, results)

		if newResultOffset := cursor.SkipResultsInDump + len(results); newResultOffset < count {
			newCursor := cursor
			newCursor.SkipResultsInDump = newResultOffset
			return resolvedLocations, newCursor, true, nil
		}

		if i+1 < len(cursor.DumpIDs) {
			newCursor := cursor
			newCursor.SkipDumpsInBatch = i + 1
			newCursor.SkipResultsInDump = 0
			return resolvedLocations, newCursor, true, nil
		}

		if cursor.SkipDumpsWhenBatching < cursor.TotalDumpsWhenBatching {
			newCursor := cursor
			newCursor.DumpIDs = []int{}
			newCursor.SkipDumpsInBatch = 0
			newCursor.SkipResultsInDump = 0
			return resolvedLocations, newCursor, true, nil
		}

		return resolvedLocations, Cursor{}, false, nil
	}

	return nil, Cursor{}, false, nil
}

func hashLocation(location bundles.Location) string {
	return fmt.Sprintf(
		"%s:%d:%d:%d:%d",
		location.Path,
		location.Range.Start.Line,
		location.Range.Start.Character,
		location.Range.End.Line,
		location.Range.End.Character,
	)
}

func applyBloomFilter(packageReferences []types.PackageReference, identifier string, limit int) ([]types.PackageReference, int) {
	var filteredReferences []types.PackageReference
	for i, ref := range packageReferences {
		test, err := bloomfilter.DecodeAndTestFilter([]byte(ref.Filter), identifier)
		if err != nil || !test {
			continue
		}

		filteredReferences = append(filteredReferences, ref)

		if len(filteredReferences) >= limit {
			return filteredReferences, i + 1
		}
	}

	return filteredReferences, len(packageReferences)
}
