package sqlite

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/keegancsmith/sqlf"
	pkgerrors "github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/cache"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/serialization"
	gobserializer "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/serialization/gob"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite/migrate"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite/store"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
)

// ErrNoMetadata occurs when there are no rows in the meta table.
var ErrNoMetadata = errors.New("no rows in meta table")

type sqliteStore struct {
	filename   string
	cache      cache.DataCache
	store      *store.Store
	closer     func() error
	serializer serialization.Serializer
}

var _ persistence.Store = &sqliteStore{}

func OpenStore(ctx context.Context, filename string, cache cache.DataCache) (_ persistence.Store, err error) {
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

	serializer := gobserializer.New()

	if err := migrate.Migrate(ctx, store, serializer); err != nil {
		return nil, err
	}

	return &sqliteStore{
		filename:   filename,
		cache:      cache,
		store:      store,
		closer:     closer,
		serializer: serializer,
	}, nil
}

func (r *sqliteStore) ReadMeta(ctx context.Context) (types.MetaData, error) {
	numResultChunks, exists, err := store.ScanFirstInt(r.store.Query(ctx, sqlf.Sprintf(
		`SELECT num_result_chunks FROM meta LIMIT 1`,
	)))
	if err != nil {
		return types.MetaData{}, err
	}
	if !exists {
		return types.MetaData{}, ErrNoMetadata
	}

	return types.MetaData{
		NumResultChunks: numResultChunks,
	}, nil
}

func (r *sqliteStore) PathsWithPrefix(ctx context.Context, prefix string) ([]string, error) {
	return store.ScanStrings(r.store.Query(ctx, sqlf.Sprintf(`SELECT path FROM documents WHERE path LIKE %s`, prefix+"%")))
}

func (r *sqliteStore) ReadDocument(ctx context.Context, path string) (types.DocumentData, bool, error) {
	key := r.makeCacheKey("document", path)
	if documentData, ok := r.getFromCache(key).(types.DocumentData); ok {
		return documentData, true, nil
	}

	data, exists, err := store.ScanFirstBytes(r.store.Query(ctx, sqlf.Sprintf(
		`SELECT data FROM documents WHERE path = %s LIMIT 1`,
		path,
	)))
	if err != nil || !exists {
		return types.DocumentData{}, false, err
	}

	documentData, err := r.serializer.UnmarshalDocumentData(data)
	if err != nil {
		return types.DocumentData{}, false, pkgerrors.Wrap(err, "serializer.UnmarshalDocumentData")
	}

	_ = r.cache.Set(key, documentData, int64(len(data)))
	return documentData, true, nil
}

func (r *sqliteStore) ReadResultChunk(ctx context.Context, id int) (types.ResultChunkData, bool, error) {
	key := r.makeCacheKey("result-chunk", strconv.Itoa(id))
	if resultChunkData, ok := r.getFromCache(key).(types.ResultChunkData); ok {
		return resultChunkData, true, nil
	}

	data, exists, err := store.ScanFirstBytes(r.store.Query(ctx, sqlf.Sprintf(
		`SELECT data FROM result_chunks WHERE id = %s LIMIT 1`,
		id,
	)))
	if err != nil || !exists {
		return types.ResultChunkData{}, false, err
	}

	resultChunkData, err := r.serializer.UnmarshalResultChunkData(data)
	if err != nil {
		return types.ResultChunkData{}, false, pkgerrors.Wrap(err, "serializer.UnmarshalResultChunkData")
	}

	_ = r.cache.Set(key, resultChunkData, int64(len(data)))
	return resultChunkData, true, nil
}

func (r *sqliteStore) ReadDefinitions(ctx context.Context, scheme, identifier string, skip, take int) ([]types.Location, int, error) {
	return r.readDefinitionReferences(ctx, "definitions", scheme, identifier, skip, take)
}

func (r *sqliteStore) ReadReferences(ctx context.Context, scheme, identifier string, skip, take int) ([]types.Location, int, error) {
	return r.readDefinitionReferences(ctx, "references", scheme, identifier, skip, take)
}

func (r *sqliteStore) readDefinitionReferences(ctx context.Context, tableName, scheme, identifier string, skip, take int) ([]types.Location, int, error) {
	locations, err := r.readMonikerLocations(ctx, tableName, scheme, identifier)
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

func (r *sqliteStore) readMonikerLocations(ctx context.Context, tableName, scheme, identifier string) ([]types.Location, error) {
	key := r.makeCacheKey(tableName, scheme, identifier)
	if locations, ok := r.getFromCache(key).([]types.Location); ok {
		return locations, nil
	}

	data, exists, err := store.ScanFirstBytes(r.store.Query(ctx, sqlf.Sprintf(
		`SELECT data FROM "`+tableName+`" WHERE scheme = %s AND identifier = %s LIMIT 1`,
		scheme,
		identifier,
	)))
	if err != nil || !exists {
		return nil, err
	}

	locations, err := r.serializer.UnmarshalLocations(data)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "serializer.UnmarshalLocations")
	}

	_ = r.cache.Set(key, locations, int64(len(data)))
	return locations, nil
}

func (r *sqliteStore) getFromCache(key string) interface{} {
	val, _ := r.cache.Get(key)
	return val
}

func (r *sqliteStore) makeCacheKey(parts ...string) string {
	return strings.Join(append(append([]string(nil), r.filename), parts...), ":")
}
