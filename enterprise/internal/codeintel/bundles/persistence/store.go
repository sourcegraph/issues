package persistence

import (
	"context"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
)

// KeyedDocumentData pairs a document with its path.
type KeyedDocumentData struct {
	Path     string
	Document types.DocumentData
}

// IndexedResultChunkData pairs a result chunk with its index.
type IndexedResultChunkData struct {
	Index       int
	ResultChunk types.ResultChunkData
}

type Store interface {
	Transact(ctx context.Context) (Store, error)
	Done(err error) error
	CreateTables(ctx context.Context) error
	Close(err error) error

	ReadMeta(ctx context.Context) (types.MetaData, error)
	PathsWithPrefix(ctx context.Context, prefix string) ([]string, error)
	ReadDocument(ctx context.Context, path string) (types.DocumentData, bool, error)
	ReadResultChunk(ctx context.Context, id int) (types.ResultChunkData, bool, error)
	ReadDefinitions(ctx context.Context, scheme, identifier string, skip, take int) ([]types.Location, int, error)
	ReadReferences(ctx context.Context, scheme, identifier string, skip, take int) ([]types.Location, int, error)

	WriteMeta(ctx context.Context, meta types.MetaData) error
	WriteDocuments(ctx context.Context, documents chan KeyedDocumentData) error
	WriteResultChunks(ctx context.Context, resultChunks chan IndexedResultChunkData) error
	WriteDefinitions(ctx context.Context, monikerLocations chan types.MonikerLocations) error
	WriteReferences(ctx context.Context, monikerLocations chan types.MonikerLocations) error
}
