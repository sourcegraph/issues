package database

import (
	"context"
	"sort"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go/ext"
	pkgerrors "github.com/pkg/errors"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client_types"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
)

// Database wraps access to a single processed bundle.
type Database interface {
	// Close closes the underlying store.
	Close() error

	// Exists determines if the path exists in the database.
	Exists(ctx context.Context, path string) (bool, error)

	// Ranges returns definition, reference, and hover data for each range within the given span of lines.
	Ranges(ctx context.Context, path string, startLine, endLine int) ([]bundles.CodeIntelligenceRange, error)

	// Definitions returns the set of locations defining the symbol at the given position.
	Definitions(ctx context.Context, path string, line, character int) ([]bundles.Location, error)

	// References returns the set of locations referencing the symbol at the given position.
	References(ctx context.Context, path string, line, character int) ([]bundles.Location, error)

	// Hover returns the hover text of the symbol at the given position.
	Hover(ctx context.Context, path string, line, character int) (string, bundles.Range, bool, error)

	// Diagnostics returns the diagnostics for the documents that have the given path prefix. This method
	// also returns the size of the complete result set to aid in pagination (along with skip and take).
	Diagnostics(ctx context.Context, prefix string, skip, take int) ([]bundles.Diagnostic, int, error)

	// MonikersByPosition returns all monikers attached ranges containing the given position. If multiple
	// ranges contain the position, then this method will return multiple sets of monikers. Each slice
	// of monikers are attached to a single range. The order of the output slice is "outside-in", so that
	// the range attached to earlier monikers enclose the range attached to later monikers.
	MonikersByPosition(ctx context.Context, path string, line, character int) ([][]bundles.MonikerData, error)

	// MonikerResults returns the locations that define or reference the given moniker. This method
	// also returns the size of the complete result set to aid in pagination (along with skip and take).
	MonikerResults(ctx context.Context, tableName, scheme, identifier string, skip, take int) ([]bundles.Location, int, error)

	// PackageInformation looks up package information data by identifier.
	PackageInformation(ctx context.Context, path string, packageInformationID string) (bundles.PackageInformationData, bool, error)
}

type databaseImpl struct {
	filename        string
	store           persistence.Store // bundle store
	numResultChunks int               // numResultChunks value from meta row
}

var _ Database = &databaseImpl{}

func newRange(startLine, startCharacter, endLine, endCharacter int) bundles.Range {
	return bundles.Range{
		Start: bundles.Position{
			Line:      startLine,
			Character: startCharacter,
		},
		End: bundles.Position{
			Line:      endLine,
			Character: endCharacter,
		},
	}
}

// DocumentPathRangeID denotes a range qualified by its containing document.
type DocumentPathRangeID struct {
	Path    string
	RangeID types.ID
}

// OpenDatabase opens a handle to the bundle file at the given path.
func OpenDatabase(ctx context.Context, filename string, store persistence.Store) (Database, error) {
	meta, err := store.ReadMeta(ctx)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "store.ReadMeta")
	}

	return &databaseImpl{
		filename:        filename,
		store:           store,
		numResultChunks: meta.NumResultChunks,
	}, nil
}

// Close closes the underlying store.
func (db *databaseImpl) Close() error {
	return db.store.Close(nil)
}

// Exists determines if the path exists in the database.
func (db *databaseImpl) Exists(ctx context.Context, path string) (bool, error) {
	_, exists, err := db.getDocumentData(ctx, path)
	return exists, pkgerrors.Wrap(err, "db.getDocumentData")
}

// Ranges returns definition, reference, and hover data for each range within the given span of lines.
func (db *databaseImpl) Ranges(ctx context.Context, path string, startLine, endLine int) ([]bundles.CodeIntelligenceRange, error) {
	documentData, exists, err := db.getDocumentData(ctx, path)
	if err != nil || !exists {
		return nil, pkgerrors.Wrap(err, "db.getDocumentData")
	}

	var rangeIDs []types.ID
	for id, r := range documentData.Ranges {
		if rangeIntersectsSpan(r, startLine, endLine) {
			rangeIDs = append(rangeIDs, id)
		}
	}

	resultIDSet := map[types.ID]struct{}{}
	for _, rangeID := range rangeIDs {
		r := documentData.Ranges[rangeID]
		resultIDSet[r.DefinitionResultID] = struct{}{}
		resultIDSet[r.ReferenceResultID] = struct{}{}
	}

	// Remove empty results
	delete(resultIDSet, "")

	var resultIDs []types.ID
	for id := range resultIDSet {
		resultIDs = append(resultIDs, id)
	}

	locations, err := db.locations(ctx, resultIDs)
	if err != nil {
		return nil, err
	}

	var codeintelRanges []bundles.CodeIntelligenceRange
	for _, rangeID := range rangeIDs {
		r := documentData.Ranges[rangeID]

		hoverText, _, err := db.hover(ctx, path, documentData, r)
		if err != nil {
			return nil, err
		}

		// Return only references that are in the same file. Otherwise this set
		// gets very big and such results are of limited use to consumers such as
		// the code intel extensions, which only use references for highlighting
		// uses of an identifier within the same file.
		fileLocalReferences := make([]bundles.Location, 0, len(locations[r.ReferenceResultID]))
		for _, r := range locations[r.ReferenceResultID] {
			if r.Path == path {
				fileLocalReferences = append(fileLocalReferences, r)
			}
		}

		codeintelRanges = append(codeintelRanges, bundles.CodeIntelligenceRange{
			Range:       newRange(r.StartLine, r.StartCharacter, r.EndLine, r.EndCharacter),
			Definitions: locations[r.DefinitionResultID],
			References:  fileLocalReferences,
			HoverText:   hoverText,
		})
	}

	sort.Slice(codeintelRanges, func(i, j int) bool {
		return compareBundleRanges(codeintelRanges[i].Range, codeintelRanges[j].Range)
	})

	return codeintelRanges, nil
}

// Definitions returns the set of locations defining the symbol at the given position.
func (db *databaseImpl) Definitions(ctx context.Context, path string, line, character int) ([]bundles.Location, error) {
	_, ranges, exists, err := db.getRangeByPosition(ctx, path, line, character)
	if err != nil || !exists {
		return nil, pkgerrors.Wrap(err, "db.getRangeByPosition")
	}

	for _, r := range ranges {
		if r.DefinitionResultID == "" {
			continue
		}

		locations, err := db.locations(ctx, []types.ID{r.DefinitionResultID})
		if err != nil {
			return nil, err
		}

		return locations[r.DefinitionResultID], nil
	}

	return []bundles.Location{}, nil
}

// References returns the set of locations referencing the symbol at the given position.
func (db *databaseImpl) References(ctx context.Context, path string, line, character int) ([]bundles.Location, error) {
	_, ranges, exists, err := db.getRangeByPosition(ctx, path, line, character)
	if err != nil || !exists {
		return nil, pkgerrors.Wrap(err, "db.getRangeByPosition")
	}

	var allLocations []bundles.Location
	for _, r := range ranges {
		if r.ReferenceResultID == "" {
			continue
		}

		locations, err := db.locations(ctx, []types.ID{r.ReferenceResultID})
		if err != nil {
			return nil, err
		}

		allLocations = append(allLocations, locations[r.ReferenceResultID]...)
	}

	return allLocations, nil
}

// Hover returns the hover text of the symbol at the given position.
func (db *databaseImpl) Hover(ctx context.Context, path string, line, character int) (string, bundles.Range, bool, error) {
	documentData, ranges, exists, err := db.getRangeByPosition(ctx, path, line, character)
	if err != nil || !exists {
		return "", bundles.Range{}, false, pkgerrors.Wrap(err, "db.getRangeByPosition")
	}

	for _, r := range ranges {
		text, exists, err := db.hover(ctx, path, documentData, r)
		if err != nil {
			return "", bundles.Range{}, false, err
		}
		if !exists {
			continue
		}

		return text, newRange(r.StartLine, r.StartCharacter, r.EndLine, r.EndCharacter), true, nil
	}

	return "", bundles.Range{}, false, nil
}

// Diagnostics returns the diagnostics for the documents that have the given path prefix. This method
// also returns the size of the complete result set to aid in pagination (along with skip and take).
func (db *databaseImpl) Diagnostics(ctx context.Context, prefix string, skip, take int) ([]bundles.Diagnostic, int, error) {
	paths, err := db.getPathsWithPrefix(ctx, prefix)
	if err != nil {
		return nil, 0, pkgerrors.Wrap(err, "db.getPathsWithPrefix")
	}

	// TODO(efritz) - we may need to store the diagnostic count outside of the
	// document so that we can efficiently skip over results that we've already
	// encountered.

	totalCount := 0
	var diagnostics []bundles.Diagnostic
	for _, path := range paths {
		documentData, exists, err := db.getDocumentData(ctx, path)
		if err != nil {
			return nil, 0, pkgerrors.Wrap(err, "db.getDocumentData")
		}
		if !exists {
			return nil, 0, nil
		}

		totalCount += len(documentData.Diagnostics)

		for _, diagnostic := range documentData.Diagnostics {
			skip--
			if skip < 0 && len(diagnostics) < take {
				diagnostics = append(diagnostics, bundles.Diagnostic{
					Path:           path,
					Severity:       diagnostic.Severity,
					Code:           diagnostic.Code,
					Message:        diagnostic.Message,
					Source:         diagnostic.Source,
					StartLine:      diagnostic.StartLine,
					StartCharacter: diagnostic.StartCharacter,
					EndLine:        diagnostic.EndLine,
					EndCharacter:   diagnostic.EndCharacter,
				})
			}
		}
	}

	return diagnostics, totalCount, nil
}

// MonikersByPosition returns all monikers attached ranges containing the given position. If multiple
// ranges contain the position, then this method will return multiple sets of monikers. Each slice
// of monikers are attached to a single range. The order of the output slice is "outside-in", so that
// the range attached to earlier monikers enclose the range attached to later monikers.
func (db *databaseImpl) MonikersByPosition(ctx context.Context, path string, line, character int) ([][]bundles.MonikerData, error) {
	documentData, ranges, exists, err := db.getRangeByPosition(ctx, path, line, character)
	if err != nil || !exists {
		return nil, pkgerrors.Wrap(err, "db.getRangeByPosition")
	}

	var monikerData [][]bundles.MonikerData
	for _, r := range ranges {
		var batch []bundles.MonikerData
		for _, monikerID := range r.MonikerIDs {
			moniker, exists := documentData.Monikers[monikerID]
			if !exists {
				log15.Warn("malformed bundle: unknown moniker", "filename", db.filename, "path", path, "id", monikerID)
				continue
			}

			batch = append(batch, bundles.MonikerData{
				Kind:                 moniker.Kind,
				Scheme:               moniker.Scheme,
				Identifier:           moniker.Identifier,
				PackageInformationID: string(moniker.PackageInformationID),
			})
		}

		monikerData = append(monikerData, batch)
	}

	return monikerData, nil
}

// MonikerResults returns the locations that define or reference the given moniker. This method
// also returns the size of the complete result set to aid in pagination (along with skip and take).
func (db *databaseImpl) MonikerResults(ctx context.Context, tableName, scheme, identifier string, skip, take int) (_ []bundles.Location, _ int, err error) {
	span, ctx := ot.StartSpanFromContext(ctx, "MonikerResults")
	span.SetTag("filename", db.filename)
	span.SetTag("tableName", tableName)
	span.SetTag("scheme", scheme)
	span.SetTag("identifier", identifier)
	defer func() {
		if err != nil {
			ext.Error.Set(span, true)
			span.SetTag("err", err.Error())
		}
		span.Finish()
	}()

	var rows []types.Location
	var totalCount int
	if tableName == "definitions" {
		if rows, totalCount, err = db.store.ReadDefinitions(ctx, scheme, identifier, skip, take); err != nil {
			err = pkgerrors.Wrap(err, "store.ReadDefinitions")
		}
	} else if tableName == "references" {
		if rows, totalCount, err = db.store.ReadReferences(ctx, scheme, identifier, skip, take); err != nil {
			err = pkgerrors.Wrap(err, "store.ReadReferences")
		}
	}

	if err != nil {
		return nil, 0, err
	}

	var locations []bundles.Location
	for _, row := range rows {
		locations = append(locations, bundles.Location{
			Path:  row.URI,
			Range: newRange(row.StartLine, row.StartCharacter, row.EndLine, row.EndCharacter),
		})
	}

	return locations, totalCount, nil
}

// PackageInformation looks up package information data by identifier.
func (db *databaseImpl) PackageInformation(ctx context.Context, path, packageInformationID string) (bundles.PackageInformationData, bool, error) {
	documentData, exists, err := db.getDocumentData(ctx, path)
	if err != nil {
		return bundles.PackageInformationData{}, false, pkgerrors.Wrap(err, "db.getDocumentData")
	}
	if !exists {
		return bundles.PackageInformationData{}, false, nil
	}

	packageInformationData, exists := documentData.PackageInformation[types.ID(packageInformationID)]
	if exists {
		return bundles.PackageInformationData{
			Name:    packageInformationData.Name,
			Version: packageInformationData.Version,
		}, true, nil
	}

	return bundles.PackageInformationData{}, false, nil
}

// hover returns the hover text locations for the given range.
func (db *databaseImpl) hover(ctx context.Context, path string, documentData types.DocumentData, r types.RangeData) (string, bool, error) {
	if r.HoverResultID == "" {
		return "", false, nil
	}

	text, exists := documentData.HoverResults[r.HoverResultID]
	if !exists {
		log15.Warn("malformed bundle: unknown hover result", "filename", db.filename, "path", path, "id", r.HoverResultID)
		return "", false, nil
	}

	return text, true, nil
}

func (db *databaseImpl) getPathsWithPrefix(ctx context.Context, prefix string) (_ []string, err error) {
	span, ctx := ot.StartSpanFromContext(ctx, "getPathsWithPrefix")
	span.SetTag("filename", db.filename)
	span.SetTag("prefix", prefix)
	defer func() {
		if err != nil {
			ext.Error.Set(span, true)
			span.SetTag("err", err.Error())
		}
		span.Finish()
	}()

	return db.store.PathsWithPrefix(ctx, prefix)
}

// getDocumentData fetches and unmarshals the document data or the given path.
func (db *databaseImpl) getDocumentData(ctx context.Context, path string) (_ types.DocumentData, _ bool, err error) {
	span, ctx := ot.StartSpanFromContext(ctx, "getDocumentData")
	span.SetTag("filename", db.filename)
	span.SetTag("path", path)
	defer func() {
		if err != nil {
			ext.Error.Set(span, true)
			span.SetTag("err", err.Error())
		}
		span.Finish()
	}()

	documentData, ok, err := db.store.ReadDocument(ctx, path)
	if err != nil {
		return types.DocumentData{}, false, pkgerrors.Wrap(err, "store.ReadDocument")
	}
	return documentData, ok, nil
}

// getRangeByPosition returns the ranges the given position. The order of the output slice is "outside-in",
// so that earlier ranges properly enclose later ranges.
func (db *databaseImpl) getRangeByPosition(ctx context.Context, path string, line, character int) (types.DocumentData, []types.RangeData, bool, error) {
	documentData, exists, err := db.getDocumentData(ctx, path)
	if err != nil {
		return types.DocumentData{}, nil, false, pkgerrors.Wrap(err, "db.getDocumentData")
	}
	if !exists {
		return types.DocumentData{}, nil, false, nil
	}

	return documentData, findRanges(documentData.Ranges, line, character), true, nil
}

// locations returns the locations for the given definition or reference identifiers.
func (db *databaseImpl) locations(ctx context.Context, ids []types.ID) (map[types.ID][]bundles.Location, error) {
	results, err := db.getResultsByIDs(ctx, ids)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "db.getResultByID")
	}

	locationWrapper, err := db.convertRangesToLocations(ctx, results)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "db.convertRangesToLocations")
	}

	return locationWrapper, nil
}

// getResultsByIDs fetches and unmarshals a definition or reference results for the given identifiers.
func (db *databaseImpl) getResultsByIDs(ctx context.Context, ids []types.ID) (map[types.ID][]DocumentPathRangeID, error) {
	xids := map[int]struct{}{}
	for _, id := range ids {
		xids[db.resultChunkID(id)] = struct{}{}
	}

	resultChunks := map[int]types.ResultChunkData{}
	for index := range xids {
		resultChunkData, exists, err := db.getResultChunkByID(ctx, index)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "db.getResultChunkByID")
		}
		if !exists {
			log15.Warn("malformed bundle: unknown result chunk", "filename", db.filename, "index", index)
			continue
		}

		resultChunks[index] = resultChunkData
	}

	data := map[types.ID][]DocumentPathRangeID{}

	for _, id := range ids {
		index := db.resultChunkID(id)
		resultChunkData := resultChunks[index]

		documentIDRangeIDs, exists := resultChunkData.DocumentIDRangeIDs[id]
		if !exists {
			log15.Warn("malformed bundle: unknown result", "filename", db.filename, "index", index, "id", id)
			continue
		}

		var resultData []DocumentPathRangeID
		for _, documentIDRangeID := range documentIDRangeIDs {
			path, ok := resultChunkData.DocumentPaths[documentIDRangeID.DocumentID]
			if !ok {
				log15.Warn("malformed bundle: unknown document path", "filename", db.filename, "index", index, "id", documentIDRangeID.DocumentID)
				continue
			}

			resultData = append(resultData, DocumentPathRangeID{
				Path:    path,
				RangeID: documentIDRangeID.RangeID,
			})
		}

		data[id] = resultData
	}

	return data, nil
}

// getResultChunkByID fetches and unmarshals the result chunk data with the given identifier.
func (db *databaseImpl) getResultChunkByID(ctx context.Context, id int) (_ types.ResultChunkData, _ bool, err error) {
	span, ctx := ot.StartSpanFromContext(ctx, "getResultChunkByID")
	span.SetTag("filename", db.filename)
	span.SetTag("id", id)
	defer func() {
		if err != nil {
			ext.Error.Set(span, true)
			span.SetTag("err", err.Error())
		}
		span.Finish()
	}()

	resultChunkData, ok, err := db.store.ReadResultChunk(ctx, id)
	if err != nil {
		return types.ResultChunkData{}, false, pkgerrors.Wrap(err, "store.ReadResultChunk")
	}
	return resultChunkData, ok, nil
}

// resultChunkID returns the identifier of the result chunk that contains the given identifier.
func (db *databaseImpl) resultChunkID(id types.ID) int {
	return types.HashKey(id, db.numResultChunks)
}

// convertRangesToLocations converts pairs of document paths and range identifiers to a list of locations.
func (db *databaseImpl) convertRangesToLocations(ctx context.Context, pairs map[types.ID][]DocumentPathRangeID) (map[types.ID][]bundles.Location, error) {
	// We potentially have to open a lot of documents. Reduce possible pressure on the
	// cache by ordering our queries so we only have to read and unmarshal each document
	// once.

	groupedResults := map[string][]types.ID{}
	for _, resultData := range pairs {
		for _, documentPathRangeID := range resultData {
			groupedResults[documentPathRangeID.Path] = append(groupedResults[documentPathRangeID.Path], documentPathRangeID.RangeID)
		}
	}

	documents := map[string]types.DocumentData{}
	for path := range groupedResults {
		documentData, exists, err := db.getDocumentData(ctx, path)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "db.getDocumentData")
		}

		if !exists {
			log15.Warn("malformed bundle: unknown document", "filename", db.filename, "path", path)
			continue
		}

		documents[path] = documentData
	}

	locationsByID := map[types.ID][]bundles.Location{}
	for id, resultData := range pairs {
		var locations []bundles.Location
		for _, documentPathRangeID := range resultData {
			path := documentPathRangeID.Path
			rangeID := documentPathRangeID.RangeID

			r, exists := documents[path].Ranges[rangeID]
			if !exists {
				log15.Warn("malformed bundle: unknown range", "filename", db.filename, "path", path, "id", id)
				continue
			}

			locations = append(locations, bundles.Location{
				Path:  path,
				Range: newRange(r.StartLine, r.StartCharacter, r.EndLine, r.EndCharacter),
			})
		}

		sort.Slice(locations, func(i, j int) bool {
			if locations[i].Path == locations[j].Path {
				return compareBundleRanges(locations[i].Range, locations[j].Range)
			}

			return strings.Compare(locations[i].Path, locations[j].Path) < 0
		})

		locationsByID[id] = locations
	}

	return locationsByID, nil
}

// compareBundleRanges returns true if r1's start position occurs before r2's start position.
func compareBundleRanges(r1, r2 bundles.Range) bool {
	cmp := r1.Start.Line - r2.Start.Line
	if cmp == 0 {
		cmp = r1.Start.Character - r2.Start.Character
	}

	return cmp < 0
}
