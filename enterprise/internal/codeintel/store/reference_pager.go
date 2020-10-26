package store

import (
	"context"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
)

// DoneFunc is the function type of store's Done method.
type DoneFunc func(err error) error

// ReferencePager holds state for a reference result within a SQL transaction so that
// each page requested has a consistent view into the underlying database.
type ReferencePager interface {
	// PageFromOffset returns the page of package references that starts at the given offset.
	PageFromOffset(ctx context.Context, offset int) ([]types.PackageReference, error)

	// Done closes the underlying transaction. If the reference pager was called on a
	// store instance that was already in a transaction, this method does nothing.
	Done(err error) error
}

// PageFromOffsetFunc is the function type of ReferencePager's PageFromOffset method.
type PageFromOffsetFunc func(ctx context.Context, offset int) ([]types.PackageReference, error)

// noopPageFromOffsetFunc is a behaviorless PageFromOffsetFunc.
func noopPageFromOffsetFunc(ctx context.Context, offset int) ([]types.PackageReference, error) {
	return nil, nil
}

// referencePager is a small struct that conforms to the ReferencePager interface.
type referencePager struct {
	pageFromOffset PageFromOffsetFunc
	done           DoneFunc
}

// PageFromOffset returns the page of package references that starts at the given offset.
func (rp *referencePager) PageFromOffset(ctx context.Context, offset int) ([]types.PackageReference, error) {
	return rp.pageFromOffset(ctx, offset)
}

// PageFromOffset returns the page of package references that starts at the given offset.
func (rp *referencePager) Done(err error) error {
	return rp.done(err)
}

func newReferencePager(pageFromOffset PageFromOffsetFunc, done DoneFunc) ReferencePager {
	return &referencePager{pageFromOffset: pageFromOffset, done: done}
}
