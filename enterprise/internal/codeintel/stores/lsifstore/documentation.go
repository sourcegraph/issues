package lsifstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/cockroachdb/errors"
	"github.com/keegancsmith/sqlf"
	"github.com/lib/pq"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/semantic"
)

// DocumentationPage returns the documentation page with the given PathID.
func (s *Store) DocumentationPage(ctx context.Context, bundleID int, pathID string) (_ *semantic.DocumentationPageData, err error) {
	ctx, _, endObservation := s.operations.documentationPage.WithAndLogger(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("bundleID", bundleID),
		log.String("pathID", pathID),
	}})
	defer endObservation(1, observation.Args{})

	page, err := s.scanFirstDocumentationPageData(s.Store.Query(ctx, sqlf.Sprintf(documentationPageDataQuery, bundleID, pathID)))
	if err != nil {
		return nil, err
	}
	return page, nil
}

const documentationPageDataQuery = `
-- source: enterprise/internal/codeintel/stores/lsifstore/documentation.go:DocumentationPage
SELECT
	dump_id,
	path_id,
	data
FROM
	lsif_data_documentation_pages
WHERE
	dump_id = %s AND
	path_id = %s
`

// scanFirstDocumentationPageData reads the first DocumentationPageData row. If no rows match the
// query, a nil is returned.
func (s *Store) scanFirstDocumentationPageData(rows *sql.Rows, queryErr error) (_ *semantic.DocumentationPageData, err error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	if !rows.Next() {
		return nil, nil
	}

	var (
		rawData  []byte
		uploadID int
		pathID   string
	)
	if err := rows.Scan(
		&uploadID,
		&pathID,
		&rawData,
	); err != nil {
		return nil, err
	}
	record, err := s.serializer.UnmarshalDocumentationPageData(rawData)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// DocumentationPathInfo returns info describing what is at the given pathID.
func (s *Store) DocumentationPathInfo(ctx context.Context, bundleID int, pathID string) (_ *semantic.DocumentationPathInfoData, err error) {
	ctx, _, endObservation := s.operations.documentationPathInfo.WithAndLogger(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("bundleID", bundleID),
		log.String("pathID", pathID),
	}})
	defer endObservation(1, observation.Args{})

	page, err := s.scanFirstDocumentationPathInfoData(s.Store.Query(ctx, sqlf.Sprintf(documentationPathInfoDataQuery, bundleID, pathID)))
	if err != nil {
		return nil, err
	}
	return page, nil
}

const documentationPathInfoDataQuery = `
-- source: enterprise/internal/codeintel/stores/lsifstore/documentation.go:DocumentationPathInfo
SELECT
	dump_id,
	path_id,
	data
FROM
	lsif_data_documentation_path_info
WHERE
	dump_id = %s AND
	path_id = %s
`

// scanFirstDocumentationPathInfoData reads the first DocumentationPathInfoData row. If no rows match the
// query, a nil is returned.
func (s *Store) scanFirstDocumentationPathInfoData(rows *sql.Rows, queryErr error) (_ *semantic.DocumentationPathInfoData, err error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	if !rows.Next() {
		return nil, nil
	}

	var (
		rawData  []byte
		uploadID int
		pathID   string
	)
	if err := rows.Scan(
		&uploadID,
		&pathID,
		&rawData,
	); err != nil {
		return nil, err
	}
	record, err := s.serializer.UnmarshalDocumentationPathInfoData(rawData)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// documentationIDsToPathIDs returns a mapping of the given documentationResult IDs to their
// associative path IDs. Empty result IDs ("") are ignored.
func (s *Store) documentationIDsToPathIDs(ctx context.Context, bundleID int, ids []semantic.ID) (_ map[semantic.ID]string, err error) {
	ctx, _, endObservation := s.operations.documentationIDsToPathIDs.WithAndLogger(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("bundleID", bundleID),
		log.String("ids", fmt.Sprint(ids)),
	}})
	defer endObservation(1, observation.Args{})

	var wantIDs []uint64
	for _, id := range ids {
		if id != "" {
			idInt, err := strconv.ParseUint(string(id), 10, 64)
			if err != nil {
				return nil, err
			}
			wantIDs = append(wantIDs, idInt)
		}
	}
	if len(wantIDs) == 0 {
		return nil, nil
	}

	pathIDs, err := s.scanDocumentationPathIDs(s.Store.Query(ctx, sqlf.Sprintf(documentationIDsToPathIDsQuery, bundleID, pq.Array(wantIDs))))
	if err != nil {
		return nil, err
	}
	return pathIDs, nil
}

// scanDocumentationPathIDs reads documentation path IDs from the given row object.
func (s *Store) scanDocumentationPathIDs(rows *sql.Rows, queryErr error) (_ map[semantic.ID]string, err error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	values := map[semantic.ID]string{}
	for rows.Next() {
		var (
			resultID uint64
			pathID   string
		)
		if err := rows.Scan(&resultID, &pathID); err != nil {
			return nil, err
		}
		values[semantic.ID(strconv.FormatUint(resultID, 10))] = pathID
	}
	return values, nil
}

const documentationIDsToPathIDsQuery = `
-- source: enterprise/internal/codeintel/stores/lsifstore/ranges.go:documentationIDsToPathIDs
SELECT
	result_id,
	path_id
FROM
	lsif_data_documentation_mappings
WHERE
	dump_id = %s AND
	result_id = ANY (%s)
`

func (s *Store) documentationPathIDToID(ctx context.Context, bundleID int, pathID string) (_ semantic.ID, err error) {
	ctx, _, endObservation := s.operations.documentationPathIDToID.WithAndLogger(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("bundleID", bundleID),
		log.String("pathID", pathID),
	}})
	defer endObservation(1, observation.Args{})

	resultID, err := s.scanFirstDocumentationResultID(s.Store.Query(ctx, sqlf.Sprintf(documentationPathIDToIDQuery, bundleID, pathID)))
	if err != nil {
		return "", err
	}
	if resultID == -1 {
		return "", err
	}
	return semantic.ID(strconv.FormatInt(resultID, 10)), nil
}

const documentationPathIDToIDQuery = `
-- source: enterprise/internal/codeintel/stores/lsifstore/ranges.go:documentationPathIDToID
SELECT
	result_id
FROM
	lsif_data_documentation_mappings
WHERE
	dump_id = %s AND
	path_id = %s
LIMIT 1
`

// scanFirstDocumentationResultID reads the first result_id row. If no rows match the query, an empty string is returned.
func (s *Store) scanFirstDocumentationResultID(rows *sql.Rows, queryErr error) (_ int64, err error) {
	if queryErr != nil {
		return -1, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	if !rows.Next() {
		return -1, nil
	}

	var resultID int64
	if err := rows.Scan(&resultID); err != nil {
		return -1, err
	}
	return resultID, nil
}

// documentationPathIDToFilePath queries the file path associated with a documentation path ID,
// e.g. the file where the documented symbol is located - if the path ID is describing such a
// symbol, or nil otherwise.
func (s *Store) documentationPathIDToFilePath(ctx context.Context, bundleID int, pathID string) (_ *string, err error) {
	ctx, _, endObservation := s.operations.documentationPathIDToFilePath.WithAndLogger(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("bundleID", bundleID),
		log.String("pathID", pathID),
	}})
	defer endObservation(1, observation.Args{})

	return s.scanFirstDocumentationFilePath(s.Store.Query(ctx, sqlf.Sprintf(documentationPathIDToFilePathQuery, bundleID, pathID)))
}

const documentationPathIDToFilePathQuery = `
-- source: enterprise/internal/codeintel/stores/lsifstore/ranges.go:documentationPathIDToFilePath
SELECT
	file_path
FROM
	lsif_data_documentation_mappings
WHERE
	dump_id = %s AND
	path_id = %s
LIMIT 1
`

// scanFirstDocumentationFilePath reads the first file_path row. If no rows match the query, an empty string is returned.
func (s *Store) scanFirstDocumentationFilePath(rows *sql.Rows, queryErr error) (_ *string, err error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	if !rows.Next() {
		return nil, nil
	}

	var filePath *string
	if err := rows.Scan(&filePath); err != nil {
		return nil, err
	}
	return filePath, nil
}

// DocumentationDefinitions returns the set of locations defining the symbol found at the given path ID, if any.
func (s *Store) DocumentationDefinitions(ctx context.Context, bundleID int, pathID string, limit, offset int) (_ []Location, _ int, err error) {
	resultID, err := s.documentationPathIDToID(ctx, bundleID, pathID)
	if err != nil || resultID == "" {
		return nil, 0, err
	}
	extractor := func(r semantic.RangeData) semantic.ID { return r.DefinitionResultID }
	operation := s.operations.documentationDefinitions
	return s.documentationDefinitionsReferences(ctx, extractor, operation, bundleID, pathID, resultID, limit, offset)
}

// DocumentationReferences returns the set of locations referencing the symbol found at the given path ID, if any.
func (s *Store) DocumentationReferences(ctx context.Context, bundleID int, pathID string, limit, offset int) (_ []Location, _ int, err error) {
	resultID, err := s.documentationPathIDToID(ctx, bundleID, pathID)
	if resultID == "" || err != nil {
		return nil, 0, err
	}
	extractor := func(r semantic.RangeData) semantic.ID { return r.ReferenceResultID }
	operation := s.operations.documentationReferences
	return s.documentationDefinitionsReferences(ctx, extractor, operation, bundleID, pathID, resultID, limit, offset)
}

func (s *Store) documentationDefinitionsReferences(
	ctx context.Context,
	extractor func(r semantic.RangeData) semantic.ID,
	operation *observation.Operation,
	bundleID int,
	pathID string,
	resultID semantic.ID,
	limit,
	offset int,
) (_ []Location, _ int, err error) {
	ctx, traceLog, endObservation := operation.WithAndLogger(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("bundleID", bundleID),
		log.String("resultID", string(resultID)),
	}})
	defer endObservation(1, observation.Args{})

	filePath, err := s.documentationPathIDToFilePath(ctx, bundleID, pathID)
	if err != nil {
		return nil, 0, errors.Wrap(err, "documentationPathIDToFilePath")
	}
	if filePath == nil {
		// The documentation result is not attached to a file, it cannot have references.
		return nil, 0, nil
	}

	documentData, exists, err := s.scanFirstDocumentData(s.Store.Query(ctx, sqlf.Sprintf(locationsDocumentQuery, bundleID, filePath)))
	if err != nil || !exists {
		return nil, 0, err
	}

	traceLog(log.Int("numRanges", len(documentData.Document.Ranges)))
	var found *semantic.RangeData
	for _, rn := range documentData.Document.Ranges {
		if rn.DocumentationResultID == resultID {
			found = &rn
			break
		}
	}
	traceLog(log.Bool("found", found == nil))
	if found == nil {
		return nil, 0, errors.New("not found")
	}

	orderedResultIDs := extractResultIDs([]semantic.RangeData{*found}, extractor)
	locationsMap, totalCount, err := s.locations(ctx, bundleID, orderedResultIDs, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	traceLog(log.Int("totalCount", totalCount))

	locations := make([]Location, 0, limit)
	for _, resultID := range orderedResultIDs {
		locations = append(locations, locationsMap[resultID]...)
	}

	return locations, totalCount, nil
}
