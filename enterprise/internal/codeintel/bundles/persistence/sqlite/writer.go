package sqlite

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/cache"
	gobserializer "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/serialization/gob"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite/batch"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite/migrate"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite/store"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
)

func NewStore(ctx context.Context, filename string, cache cache.DataCache) (_ persistence.Store, err error) {
	store, closer, err := store.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if closeErr := closer(); closeErr != nil {
				err = multierror.Append(err, closeErr)
			}
		}
	}()

	return &sqliteStore{
		filename:   filename,
		cache:      cache,
		store:      store,
		closer:     closer,
		serializer: gobserializer.New(),
	}, nil
}

func (w *sqliteStore) Transact(ctx context.Context) (persistence.Store, error) {
	tx, err := w.store.Transact(ctx)
	if err != nil {
		return nil, err
	}

	return &sqliteStore{
		store:      tx,
		closer:     w.closer,
		serializer: w.serializer,
	}, nil
}

func (w *sqliteStore) Done(err error) error {
	return w.store.Done(err)
}

func (w *sqliteStore) WriteMeta(ctx context.Context, metaData types.MetaData) error {
	queries := []*sqlf.Query{
		sqlf.Sprintf("INSERT INTO schema_version (version) VALUES (%s)", migrate.CurrentSchemaVersion),
		sqlf.Sprintf("INSERT INTO meta (num_result_chunks) VALUES (%s)", metaData.NumResultChunks),
	}

	for _, query := range queries {
		if err := w.store.Exec(ctx, query); err != nil {
			return err
		}
	}

	return nil
}

func (w *sqliteStore) WriteDocuments(ctx context.Context, documents chan persistence.KeyedDocumentData) error {
	return batch.WriteDocuments(ctx, w.store, "documents", w.serializer, documents)
}

func (w *sqliteStore) WriteResultChunks(ctx context.Context, resultChunks chan persistence.IndexedResultChunkData) error {
	return batch.WriteResultChunks(ctx, w.store, "result_chunks", w.serializer, resultChunks)
}

func (w *sqliteStore) WriteDefinitions(ctx context.Context, monikerLocations chan types.MonikerLocations) error {
	return batch.WriteMonikerLocations(ctx, w.store, "definitions", w.serializer, monikerLocations)
}

func (w *sqliteStore) WriteReferences(ctx context.Context, monikerLocations chan types.MonikerLocations) error {
	return batch.WriteMonikerLocations(ctx, w.store, "references", w.serializer, monikerLocations)
}

func (w *sqliteStore) Close(err error) error {
	if closeErr := w.closer(); closeErr != nil {
		err = multierror.Append(err, closeErr)
	}

	return err
}

func (w *sqliteStore) CreateTables(ctx context.Context) error {
	queries := []*sqlf.Query{
		sqlf.Sprintf(`CREATE TABLE "schema_version" ("version" text NOT NULL)`),
		sqlf.Sprintf(`CREATE TABLE "meta" ("num_result_chunks" integer NOT NULL)`),
		sqlf.Sprintf(`CREATE TABLE "documents" ("path" text PRIMARY KEY NOT NULL, "data" blob NOT NULL)`),
		sqlf.Sprintf(`CREATE TABLE "result_chunks" ("id" integer PRIMARY KEY NOT NULL, "data" blob NOT NULL)`),
		sqlf.Sprintf(`CREATE TABLE "definitions" ("scheme" text NOT NULL, "identifier" text NOT NULL, "data" blob NOT NULL, PRIMARY KEY (scheme, identifier))`),
		sqlf.Sprintf(`CREATE TABLE "references" ("scheme" text NOT NULL, "identifier" text NOT NULL, "data" blob NOT NULL, PRIMARY KEY (scheme, identifier))`),
	}

	for _, query := range queries {
		if err := w.store.Exec(ctx, query); err != nil {
			return err
		}
	}

	return nil
}
