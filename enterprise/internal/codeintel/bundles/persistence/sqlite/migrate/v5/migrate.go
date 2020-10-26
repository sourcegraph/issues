package v5

import (
	"context"
	"fmt"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/serialization"
	jsonserializer "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/serialization/json"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite/batch"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite/store"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite/util"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
)

type MigrationStep func(context.Context, *store.Store, serialization.Serializer, serialization.Serializer) error

// Migrate v5: updates the serialization format of data blobs in the document, result chunks, definition,
// and reference tables. For each table, we create a new temporary table with the same columns, scan the
// rows in the original table, decode them with the old serializer, re-encode them with the new serializer,
// then write the new rows into the temporary table. The original table is dropped and the temporary table
// renamed to take its place.
func Migrate(ctx context.Context, s *store.Store, serializer serialization.Serializer) error {
	steps := []MigrationStep{
		createTempTables,
		reencodeDocuments,
		reencodeResultChunks,
		reencodeDefinitions,
		reencodeReferences,
		swapTables,
	}

	// NOTE: We need to serialize with the JSON serializer, NOT the current serializer. This is
	// because future migrations assume that v4 was written with the most current serializer at
	// that time. Using the current serializer will cause future migrations to fail to read the
	// encoded data.
	deserializer := jsonserializer.New()

	for _, step := range steps {
		if err := step(ctx, s, deserializer, serializer); err != nil {
			return err
		}
	}

	return nil
}

// createTempTables creates new, empty tables that will be the target for the new serialization format.
func createTempTables(ctx context.Context, s *store.Store, _, _ serialization.Serializer) error {
	queries := []*sqlf.Query{
		sqlf.Sprintf(`CREATE TABLE "t_documents" ("path" text PRIMARY KEY NOT NULL, "data" blob NOT NULL)`),
		sqlf.Sprintf(`CREATE TABLE "t_result_chunks" ("id" integer PRIMARY KEY NOT NULL, "data" blob NOT NULL)`),
		sqlf.Sprintf(`CREATE TABLE "t_definitions" ("scheme" text NOT NULL, "identifier" text NOT NULL, "data" blob NOT NULL, PRIMARY KEY (scheme, identifier))`),
		sqlf.Sprintf(`CREATE TABLE "t_references" ("scheme" text NOT NULL, "identifier" text NOT NULL, "data" blob NOT NULL, PRIMARY KEY (scheme, identifier))`),
	}

	for _, query := range queries {
		if err := s.Exec(ctx, query); err != nil {
			return err
		}
	}

	return nil
}

// reencodeDocuments pulls data from the old document table and inserts the re-encoded data into the temporary table.
func reencodeDocuments(ctx context.Context, s *store.Store, deserializer, serializer serialization.Serializer) error {
	ch := make(chan persistence.KeyedDocumentData)

	return util.InvokeAll(
		func() error { return readDocuments(ctx, s, deserializer, ch) },
		func() error { return batch.WriteDocumentsChan(ctx, s, "t_documents", serializer, ch) },
	)
}

// reencodeResultChunks pulls data from the old result chunks table and inserts the re-encoded data into the temporary table.
func reencodeResultChunks(ctx context.Context, s *store.Store, deserializer, serializer serialization.Serializer) error {
	ch := make(chan persistence.IndexedResultChunkData)

	return util.InvokeAll(
		func() error { return readResultChunks(ctx, s, deserializer, ch) },
		func() error { return batch.WriteResultChunksChan(ctx, s, "t_result_chunks", serializer, ch) },
	)
}

// reencodeResultChunks pulls data from the old definitions table and inserts the re-encoded data into the temporary table.
func reencodeDefinitions(ctx context.Context, s *store.Store, deserializer, serializer serialization.Serializer) error {
	return reencodeDefinitionReferences(ctx, s, "definitions", deserializer, serializer)
}

// reencodeResultChunks pulls data from the old references table and inserts the re-encoded data into the temporary table.
func reencodeReferences(ctx context.Context, s *store.Store, deserializer, serializer serialization.Serializer) error {
	return reencodeDefinitionReferences(ctx, s, "references", deserializer, serializer)
}

// reencodeResultChunks pulls data from the old definitions or references table and inserts the re-encoded data into the temporary table.
func reencodeDefinitionReferences(ctx context.Context, s *store.Store, tableName string, deserializer, serializer serialization.Serializer) (err error) {
	ch := make(chan types.MonikerLocations)
	tempTableName := fmt.Sprintf("t_%s", tableName)

	return util.InvokeAll(
		func() error { return readMonikerLocations(ctx, s, tableName, deserializer, ch) },
		func() error { return batch.WriteMonikerLocationsChan(ctx, s, tempTableName, serializer, ch) },
	)
}

// swapTables deletes the originals table and replaces them with the tables containing data in the serialization format.
func swapTables(ctx context.Context, s *store.Store, _, _ serialization.Serializer) error {
	queries := []*sqlf.Query{
		sqlf.Sprintf(`DROP TABLE "documents"`),
		sqlf.Sprintf(`ALTER TABLE "t_documents" RENAME TO "documents"`),
		sqlf.Sprintf(`DROP TABLE "result_chunks"`),
		sqlf.Sprintf(`ALTER TABLE "t_result_chunks" RENAME TO "result_chunks"`),
		sqlf.Sprintf(`DROP TABLE "definitions"`),
		sqlf.Sprintf(`ALTER TABLE "t_definitions" RENAME TO "definitions"`),
		sqlf.Sprintf(`DROP TABLE "references"`),
		sqlf.Sprintf(`ALTER TABLE "t_references" RENAME TO "references"`),
	}

	for _, query := range queries {
		if err := s.Exec(ctx, query); err != nil {
			return err
		}
	}

	return nil
}

// readDocuments reads all documents from the original documents table and writes the scanned results onto the
// given channel. If an error occurs during query or scanning, that error is returned and no future writes to
// the channel will be performed. The given channel is closed when the function exits.
func readDocuments(ctx context.Context, s *store.Store, serializer serialization.Serializer, ch chan<- persistence.KeyedDocumentData) (err error) {
	defer close(ch)

	rows, err := s.Query(ctx, sqlf.Sprintf("SELECT path, data FROM documents"))
	if err != nil {
		return err
	}
	defer func() { err = store.CloseRows(rows, err) }()

	for rows.Next() {
		var path string
		var data []byte
		if err := rows.Scan(&path, &data); err != nil {
			return err
		}

		document, err := serializer.UnmarshalDocumentData(data)
		if err != nil {
			return err
		}

		ch <- persistence.KeyedDocumentData{
			Path:     path,
			Document: document,
		}
	}

	return nil
}

// readResultChunks reads all result chunks from the original result chunks table and writes the scanned results
// onto the given channel. If an error occurs during query or scanning, that error is returned and no future writes
// to the channel will be performed. The given channel is closed when the function exits.
func readResultChunks(ctx context.Context, s *store.Store, serializer serialization.Serializer, ch chan<- persistence.IndexedResultChunkData) (err error) {
	defer close(ch)

	rows, err := s.Query(ctx, sqlf.Sprintf("SELECT id, data FROM result_chunks"))
	if err != nil {
		return err
	}
	defer func() { err = store.CloseRows(rows, err) }()

	for rows.Next() {
		var id int
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return err
		}

		resultChunk, err := serializer.UnmarshalResultChunkData(data)
		if err != nil {
			return err
		}

		ch <- persistence.IndexedResultChunkData{
			Index:       id,
			ResultChunk: resultChunk,
		}
	}

	return nil
}

// readMonikerLocations reads all moniker locations from the given table and writes the scanned results onto the given
// channel. If an error occurs during query or scanning, that error is returned and no future writes to the channel will
// be performed. The given channel is closed when the function exits.
func readMonikerLocations(ctx context.Context, s *store.Store, tableName string, serializer serialization.Serializer, ch chan<- types.MonikerLocations) (err error) {
	defer close(ch)

	rows, err := s.Query(ctx, sqlf.Sprintf(`SELECT scheme, identifier, data FROM "`+tableName+`"`))
	if err != nil {
		return err
	}
	defer func() { err = store.CloseRows(rows, err) }()

	for rows.Next() {
		var scheme string
		var identifier string
		var data []byte
		if err := rows.Scan(&scheme, &identifier, &data); err != nil {
			return err
		}

		locations, err := serializer.UnmarshalLocations(data)
		if err != nil {
			return err
		}

		ch <- types.MonikerLocations{
			Scheme:     scheme,
			Identifier: identifier,
			Locations:  locations,
		}
	}

	return nil
}
