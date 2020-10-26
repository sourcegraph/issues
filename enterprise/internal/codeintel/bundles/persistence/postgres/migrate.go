package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/keegancsmith/sqlf"
	"github.com/pkg/errors"
	sqlitestore "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite/store"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/db/batch"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

// MigrateBundleToPostgres reads the SQLite file at the given filename and inserts the same
// data into the given Postgres handle. Every row inserted into Postgres will be inserted with
// the given dump identifier.
func MigrateBundleToPostgres(ctx context.Context, dumpID int, filename string, to dbutil.DB) (err error) {
	from, closer, err := sqlitestore.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closer(); closeErr != nil {
			err = multierror.Append(err, closeErr)
		}
	}()

	if err := migrateMeta(ctx, from, to, dumpID); err != nil {
		return errors.Wrap(err, "migrating meta")
	}
	if err := migrateDocuments(ctx, from, to, dumpID); err != nil {
		return errors.Wrap(err, "migrating documents")
	}
	if err := migrateResultChunks(ctx, from, to, dumpID); err != nil {
		return errors.Wrap(err, "migrating result chunks")
	}
	if err := migrateDefinitions(ctx, from, to, dumpID); err != nil {
		return errors.Wrap(err, "migrating definitions")
	}
	if err := migrateReferences(ctx, from, to, dumpID); err != nil {
		return errors.Wrap(err, "migrating references")
	}

	return nil
}

func migrateMeta(ctx context.Context, from *sqlitestore.Store, to dbutil.DB, dumpID int) error {
	values, err := sqlitestore.ScanInts(from.Query(ctx, sqlf.Sprintf(`SELECT num_result_chunks FROM meta`)))
	if err != nil {
		return err
	}

	ch := make(chan int, 1)
	ch <- values[0]
	close(ch)

	// There should only be one row to insert here. withBatchInserter assumes that
	// the given inserter function can be called from multiple goroutines, so we feed
	// the single value into a channel so that only one of the invocations will write
	// the row to the database.
	return withBatchInserter(ctx, to, "lsif_data_metadata", []string{"dump_id", "num_result_chunks"}, func(inserter *batch.BatchInserter) error {
		for v := range ch {
			if err := inserter.Insert(ctx, dumpID, v); err != nil {
				return err
			}
		}

		return nil
	})
}

type document struct {
	path string
	data string
	err  error
}

func migrateDocuments(ctx context.Context, from *sqlitestore.Store, to dbutil.DB, dumpID int) error {
	ch := make(chan document)
	go func() {
		defer close(ch)

		scan := func(rows *sql.Rows) error {
			var value document
			if err := rows.Scan(&value.path, &value.data); err != nil {
				return err
			}

			ch <- value
			return nil
		}

		if err := foreachRow(ctx, from, sqlf.Sprintf(`SELECT path, data FROM documents`), scan); err != nil {
			ch <- document{err: err}
		}
	}()

	return withBatchInserter(ctx, to, "lsif_data_documents", []string{"dump_id", "path", "data"}, func(inserter *batch.BatchInserter) error {
		for document := range ch {
			if document.err != nil {
				return document.err
			}

			if err := inserter.Insert(ctx, dumpID, document.path, document.data); err != nil {
				return err
			}
		}

		return nil
	})
}

type resultChunk struct {
	index int
	data  string
	err   error
}

func migrateResultChunks(ctx context.Context, from *sqlitestore.Store, to dbutil.DB, dumpID int) error {
	ch := make(chan resultChunk)
	go func() {
		defer close(ch)

		scan := func(rows *sql.Rows) error {
			var value resultChunk
			if err := rows.Scan(&value.index, &value.data); err != nil {
				return err
			}

			ch <- value
			return nil
		}

		if err := foreachRow(ctx, from, sqlf.Sprintf(`SELECT id, data FROM result_chunks`), scan); err != nil {
			ch <- resultChunk{err: err}
		}
	}()

	return withBatchInserter(ctx, to, "lsif_data_result_chunks", []string{"dump_id", "idx", "data"}, func(inserter *batch.BatchInserter) error {
		for resultChunk := range ch {
			if resultChunk.err != nil {
				return resultChunk.err
			}

			if err := inserter.Insert(ctx, dumpID, resultChunk.index, resultChunk.data); err != nil {
				return err
			}
		}

		return nil
	})
}

func migrateDefinitions(ctx context.Context, from *sqlitestore.Store, to dbutil.DB, dumpID int) (err error) {
	return migrateDefinitionReferences(ctx, "definitions", from, to, dumpID)
}

func migrateReferences(ctx context.Context, from *sqlitestore.Store, to dbutil.DB, dumpID int) (err error) {
	return migrateDefinitionReferences(ctx, "references", from, to, dumpID)
}

type location struct {
	scheme     string
	identifier string
	data       string
	err        error
}

func migrateDefinitionReferences(ctx context.Context, tableName string, from *sqlitestore.Store, to dbutil.DB, dumpID int) (err error) {
	ch := make(chan location)
	go func() {
		defer close(ch)

		scan := func(rows *sql.Rows) error {
			var value location
			if err := rows.Scan(&value.scheme, &value.identifier, &value.data); err != nil {
				return err
			}

			ch <- value
			return nil
		}

		if err := foreachRow(ctx, from, sqlf.Sprintf(`SELECT scheme, identifier, data FROM "`+tableName+`"`), scan); err != nil {
			ch <- location{err: err}
		}
	}()

	return withBatchInserter(ctx, to, fmt.Sprintf("lsif_data_%s", tableName), []string{"dump_id", "scheme", "identifier", "data"}, func(inserter *batch.BatchInserter) error {
		for location := range ch {
			if location.err != nil {
				return location.err
			}

			if err := inserter.Insert(ctx, dumpID, location.scheme, location.identifier, location.data); err != nil {
				return err
			}
		}

		return nil
	})
}

// foreachRow performs the given query on the SQLite store and invokes the given function with
// each resulting row. The first error encountered will be returned.
func foreachRow(ctx context.Context, store *sqlitestore.Store, query *sqlf.Query, f func(rows *sql.Rows) error) (err error) {
	rows, err := store.Query(ctx, query)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := basestore.CloseRows(rows, nil); closeErr != nil {
			err = multierror.Append(err, closeErr)
		}
	}()

	for rows.Next() {
		if err := f(rows); err != nil {
			return err
		}
	}

	return nil
}
