package postgres

import (
	"context"
	"database/sql"
	"runtime"

	"github.com/hashicorp/go-multierror"
	"github.com/keegancsmith/sqlf"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/serialization"
	gobserializer "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/serialization/gob"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite/util"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/db/batch"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

var ErrNoMetadata = errors.New("no rows in meta table")

type store struct {
	*basestore.Store
	dumpID     int
	serializer serialization.Serializer
}

var _ persistence.Store = &store{}

func NewStore(db dbutil.DB, dumpID int) persistence.Store {
	return &store{
		Store:      basestore.NewWithHandle(basestore.NewHandleWithDB(db, sql.TxOptions{})),
		dumpID:     dumpID,
		serializer: gobserializer.New(),
	}
}

func (s *store) Transact(ctx context.Context) (persistence.Store, error) {
	tx, err := s.Store.Transact(ctx)
	if err != nil {
		return nil, err
	}

	return &store{
		Store:      tx,
		dumpID:     s.dumpID,
		serializer: s.serializer,
	}, nil
}

func (s *store) Done(err error) error {
	return s.Store.Done(err)
}

func (s *store) CreateTables(ctx context.Context) error {
	return nil // no-op
}

func (s *store) Close(err error) error {
	return err
}

func (s *store) ReadMeta(ctx context.Context) (types.MetaData, error) {
	numResultChunks, ok, err := basestore.ScanFirstInt(s.Store.Query(
		ctx,
		sqlf.Sprintf(
			`SELECT num_result_chunks FROM lsif_data_metadata WHERE dump_id = %s`,
			s.dumpID,
		),
	))
	if err != nil {
		return types.MetaData{}, err
	}
	if !ok {
		return types.MetaData{}, ErrNoMetadata
	}

	return types.MetaData{NumResultChunks: numResultChunks}, nil
}

func (s *store) PathsWithPrefix(ctx context.Context, prefix string) ([]string, error) {
	paths, err := basestore.ScanStrings(s.Store.Query(
		ctx,
		sqlf.Sprintf(
			`SELECT path FROM lsif_data_documents WHERE dump_id = %s AND path LIKE %s ORDER BY path`,
			s.dumpID,
			prefix+"%",
		),
	))
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (s *store) ReadDocument(ctx context.Context, path string) (types.DocumentData, bool, error) {
	data, ok, err := basestore.ScanFirstString(s.Store.Query(
		ctx,
		sqlf.Sprintf(
			`SELECT data FROM lsif_data_documents WHERE dump_id = %s AND path = %s LIMIT 1`,
			s.dumpID,
			path,
		),
	))
	if err != nil || !ok {
		return types.DocumentData{}, false, err
	}

	documentData, err := s.serializer.UnmarshalDocumentData([]byte(data))
	if err != nil {
		return types.DocumentData{}, false, err
	}

	return documentData, true, nil
}

func (s *store) ReadResultChunk(ctx context.Context, id int) (types.ResultChunkData, bool, error) {
	data, ok, err := basestore.ScanFirstString(s.Store.Query(
		ctx,
		sqlf.Sprintf(
			`SELECT data FROM lsif_data_result_chunks WHERE dump_id = %s AND idx = %s LIMIT 1`,
			s.dumpID,
			id,
		),
	))
	if err != nil || !ok {
		return types.ResultChunkData{}, false, err
	}

	resultChunkData, err := s.serializer.UnmarshalResultChunkData([]byte(data))
	if err != nil {
		return types.ResultChunkData{}, false, err
	}

	return resultChunkData, true, nil
}

func (s *store) ReadDefinitions(ctx context.Context, scheme, identifier string, skip, take int) ([]types.Location, int, error) {
	return s.readDefinitionReferences(ctx, "lsif_data_definitions", scheme, identifier, skip, take)
}

func (s *store) ReadReferences(ctx context.Context, scheme, identifier string, skip, take int) ([]types.Location, int, error) {
	return s.readDefinitionReferences(ctx, "lsif_data_references", scheme, identifier, skip, take)
}

func (s *store) readDefinitionReferences(ctx context.Context, tableName, scheme, identifier string, skip, take int) ([]types.Location, int, error) {
	data, ok, err := basestore.ScanFirstString(s.Store.Query(
		ctx,
		sqlf.Sprintf(
			`SELECT data FROM "`+tableName+`" WHERE dump_id = %s AND scheme = %s AND identifier = %s LIMIT 1`,
			s.dumpID,
			scheme,
			identifier,
		),
	))
	if err != nil || !ok {
		return nil, 0, err
	}

	locations, err := s.serializer.UnmarshalLocations([]byte(data))
	if err != nil {
		return nil, 0, err
	}

	if skip == 0 && take == 0 {
		// Pagination is disabled, return full result set
		return locations, len(locations), nil
	}

	lo := skip
	if lo >= len(locations) {
		// Skip lands past result set, return nothing
		return nil, len(locations), nil
	}

	hi := skip + take
	if hi >= len(locations) {
		hi = len(locations)
	}

	return locations[lo:hi], len(locations), nil
}

func (s *store) WriteMeta(ctx context.Context, meta types.MetaData) (err error) {
	inserter := batch.NewBatchInserter(ctx, s.Handle().DB(), "lsif_data_metadata", "dump_id", "num_result_chunks")

	defer func() {
		if flushErr := inserter.Flush(ctx); flushErr != nil {
			err = multierror.Append(err, errors.Wrap(flushErr, "inserter.Flush"))
		}
	}()

	return inserter.Insert(ctx, s.dumpID, meta.NumResultChunks)
}

func (s *store) WriteDocuments(ctx context.Context, documents chan persistence.KeyedDocumentData) error {
	inserter := func(inserter *batch.BatchInserter) error {
		for v := range documents {
			data, err := s.serializer.MarshalDocumentData(v.Document)
			if err != nil {
				return err
			}

			if err := inserter.Insert(ctx, s.dumpID, v.Path, data); err != nil {
				return err
			}
		}

		return nil
	}

	return withBatchInserter(ctx, s.Handle().DB(), "lsif_data_documents", []string{"dump_id", "path", "data"}, inserter)
}

func (s *store) WriteResultChunks(ctx context.Context, resultChunks chan persistence.IndexedResultChunkData) error {
	inserter := func(inserter *batch.BatchInserter) error {
		for v := range resultChunks {
			data, err := s.serializer.MarshalResultChunkData(v.ResultChunk)
			if err != nil {
				return err
			}

			if err := inserter.Insert(ctx, s.dumpID, v.Index, data); err != nil {
				return err
			}
		}

		return nil
	}

	return withBatchInserter(ctx, s.Handle().DB(), "lsif_data_result_chunks", []string{"dump_id", "idx", "data"}, inserter)
}

func (s *store) WriteDefinitions(ctx context.Context, monikerLocations chan types.MonikerLocations) error {
	return s.writeDefinitionReferences(ctx, "lsif_data_definitions", monikerLocations)
}

func (s *store) WriteReferences(ctx context.Context, monikerLocations chan types.MonikerLocations) error {
	return s.writeDefinitionReferences(ctx, "lsif_data_references", monikerLocations)
}

func (s *store) writeDefinitionReferences(ctx context.Context, tableName string, monikerLocations chan types.MonikerLocations) error {
	inserter := func(inserter *batch.BatchInserter) error {
		for v := range monikerLocations {
			data, err := s.serializer.MarshalLocations(v.Locations)
			if err != nil {
				return err
			}

			if err := inserter.Insert(ctx, s.dumpID, v.Scheme, v.Identifier, data); err != nil {
				return err
			}
		}

		return nil
	}

	return withBatchInserter(ctx, s.Handle().DB(), tableName, []string{"dump_id", "scheme", "identifier", "data"}, inserter)
}

var numWriterRoutines = runtime.GOMAXPROCS(0)

func withBatchInserter(ctx context.Context, db dbutil.DB, tableName string, columns []string, f func(inserter *batch.BatchInserter) error) error {
	return util.InvokeN(numWriterRoutines, func() (err error) {
		inserter := batch.NewBatchInserter(ctx, db, tableName, columns...)

		defer func() {
			if flushErr := inserter.Flush(ctx); flushErr != nil {
				err = multierror.Append(err, errors.Wrap(flushErr, "inserter.Flush"))
			}
		}()

		return f(inserter)
	})
}
