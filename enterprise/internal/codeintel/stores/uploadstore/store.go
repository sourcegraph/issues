package uploadstore

import (
	"context"
	"io"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/observation"
)

// Store is an expiring key/value store backed by a managed blob store.
type Store interface {
	// Init ensures that the underlying target bucket exists and has the expected ACL
	// and lifecycle configuration.
	Init(ctx context.Context) error

	// Get returns a reader that streams the content of the object at the given key.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Upload writes the content in the given reader to the object at the given key.
	Upload(ctx context.Context, key string, r io.Reader) (int64, error)

	// Compose will concatenate the given source objects together and write to the given
	// destination object. The source objects will be removed if the composed write is
	// successful.
	Compose(ctx context.Context, destination string, sources ...string) (int64, error)

	// Delete removes the content at the given key.
	Delete(ctx context.Context, key string) error
}

var storeConstructors = map[string]func(ctx context.Context, config *Config, operations *operations) (Store, error){
	"s3":    newS3FromConfig,
	"minio": newS3FromConfig,
	"gcs":   newGCSFromConfig,
}

// CreateLazy initialize a new store from the given configuration that is initialized
// on it first method call. If initialization fails, all methods calls will return a
// the initialization error.
func CreateLazy(ctx context.Context, config *Config, observationContext *observation.Context) (Store, error) {
	store, err := create(ctx, config, observationContext)
	if err != nil {
		return nil, err
	}

	return newLazyStore(store), nil
}

// create creates but does not initialize a new store from the given configuration.
func create(ctx context.Context, config *Config, observationContext *observation.Context) (Store, error) {
	newStore, ok := storeConstructors[config.Backend]
	if !ok {
		return nil, errors.Errorf("unknown upload store backend '%s'", config.Backend)
	}

	store, err := newStore(ctx, config, newOperations(observationContext))
	if err != nil {
		return nil, err
	}

	return store, nil
}
