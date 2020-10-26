package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

// Store is the interface to Postgres for precise-code-intel features.
type Store interface {
	// Handle returns the underlying transactable database handle.
	Handle() *basestore.TransactableHandle

	// With creates a new store with the underlying database handle from the given store.
	With(other basestore.ShareableStore) Store

	// Transact returns a store whose methods operate within the context of a transaction.
	// This method will return an error if the underlying store cannot be interface upgraded
	// to a TxBeginner.
	Transact(ctx context.Context) (Store, error)

	// Done commits underlying the transaction on a nil error value and performs a rollback
	// otherwise. If an error occurs during commit or rollback of the transaction, the error
	// is added to the resulting error value. If the store does not wrap a transaction the
	// original error value is returned unchanged.
	Done(err error) error

	// Lock attempts to take an advisory lock on the given key. If successful, this method will
	// return a true-valued flag along with a function that must be called to release the lock.
	Lock(ctx context.Context, key int, blocking bool) (bool, UnlockFunc, error)

	// GetUploadByID returns an upload by its identifier and boolean flag indicating its existence.
	GetUploadByID(ctx context.Context, id int) (Upload, bool, error)

	// GetUploads returns a list of uploads and the total count of records matching the given conditions.
	GetUploads(ctx context.Context, opts GetUploadsOptions) ([]Upload, int, error)

	// QueueSize returns the number of uploads in the queued state.
	QueueSize(ctx context.Context) (int, error)

	// InsertUpload inserts a new upload and returns its identifier.
	InsertUpload(ctx context.Context, upload Upload) (int, error)

	// AddUploadPart adds the part index to the given upload's uploaded parts array. This method is idempotent
	// (the resulting array is deduplicated on update).
	AddUploadPart(ctx context.Context, uploadID, partIndex int) error

	// MarkQueued updates the state of the upload to queued and updates the upload size.
	MarkQueued(ctx context.Context, uploadID int, uploadSize *int) error

	// MarkComplete updates the state of the upload to complete.
	MarkComplete(ctx context.Context, id int) error

	// MarkErrored updates the state of the upload to errored and updates the failure summary data.
	MarkErrored(ctx context.Context, id int, failureMessage string) error

	// Dequeue selects the oldest queued upload smaller than the given maximum size and locks it with a transaction.
	// If there is such an upload, the upload is returned along with a store instance which wraps the transaction.
	// This transaction must be closed. If there is no such unlocked upload, a zero-value upload and nil store will
	// be returned along with a false valued flag. This method must not be called from within a transaction.
	Dequeue(ctx context.Context, maxSize int64) (Upload, Store, bool, error)

	// Requeue updates the state of the upload to queued and adds a processing delay before the next dequeue attempt.
	Requeue(ctx context.Context, id int, after time.Time) error

	// GetStates returns the states for the uploads with the given identifiers.
	GetStates(ctx context.Context, ids []int) (map[int]string, error)

	// DeleteUploadByID deletes an upload by its identifier. This method returns a true-valued flag if a record
	// was deleted. The associated repository will be marked as dirty so that its commit graph will be updated in
	// the background.
	DeleteUploadByID(ctx context.Context, id int) (bool, error)

	// DeleteUploadsWithoutRepository deletes uploads associated with repositories that were deleted at least
	// DeletedRepositoryGracePeriod ago. This returns the repository identifier mapped to the number of uploads
	// that were removed for that repository.
	DeleteUploadsWithoutRepository(ctx context.Context, now time.Time) (map[int]int, error)

	// HardDeleteUploadByID deletes the upload record with the given identifier.
	HardDeleteUploadByID(ctx context.Context, ids ...int) error

	// ResetStalled moves all unlocked uploads processing for more than `StalledUploadMaxAge` back to the queued state.
	// In order to prevent input that continually crashes worker instances, uploads that have been reset more than
	// UploadMaxNumResets times will be marked as errored. This method returns a list of updated and errored upload
	// identifiers.
	ResetStalled(ctx context.Context, now time.Time) ([]int, []int, error)

	// GetDumpByID returns a dump by its identifier and boolean flag indicating its existence.
	GetDumpByID(ctx context.Context, id int) (Dump, bool, error)

	// FindClosestDumps returns the set of dumps that can most accurately answer queries for the given repository, commit, path, and
	// optional indexer. If rootMustEnclosePath is true, then only dumps with a root which is a prefix of path are returned. Otherwise,
	// any dump with a root intersecting the given path is returned.
	FindClosestDumps(ctx context.Context, repositoryID int, commit, path string, rootMustEnclosePath bool, indexer string) ([]Dump, error)

	// FindClosestDumpsFromGraphFragment returns the set of dumps that can most accurately answer queries for the given repository, commit,
	// path, and optional indexer by only considering the given fragment of the full git graph. See FindClosestDumps for additional details.
	FindClosestDumpsFromGraphFragment(ctx context.Context, repositoryID int, commit, path string, rootMustEnclosePath bool, indexer string, graph map[string][]string) ([]Dump, error)

	// DeleteOldestDump deletes the oldest dump that is not currently visible at the tip of its repository's default branch.
	// This method returns the deleted dump's identifier and a flag indicating its (previous) existence. The associated repository
	// will be marked as dirty so that its commit graph will be updated in the background.
	DeleteOldestDump(ctx context.Context) (int, bool, error)

	// SoftDeleteOldDumps marks dumps older than the given age that are not visible at the tip of the default branch
	// as deleted. The associated repositories will be marked as dirty so that their commit graphs are updated in the
	// background.
	SoftDeleteOldDumps(ctx context.Context, maxAge time.Duration, now time.Time) (int, error)

	// DeleteOverlapapingDumps deletes all completed uploads for the given repository with the same
	// commit, root, and indexer. This is necessary to perform during conversions before changing
	// the state of a processing upload to completed as there is a unique index on these four columns.
	DeleteOverlappingDumps(ctx context.Context, repositoryID int, commit, root, indexer string) error

	// GetPackage returns the dump that provides the package with the given scheme, name, and version and a flag indicating its existence.
	GetPackage(ctx context.Context, scheme, name, version string) (Dump, bool, error)

	// UpdatePackages bulk upserts package data.
	UpdatePackages(ctx context.Context, packages []types.Package) error

	// SameRepoPager returns a ReferencePager for dumps that belong to the given repository and commit and reference the package with the
	// given scheme, name, and version.
	SameRepoPager(ctx context.Context, repositoryID int, commit, scheme, name, version string, limit int) (int, ReferencePager, error)

	// UpdatePackageReferences bulk inserts package reference data.
	UpdatePackageReferences(ctx context.Context, packageReferences []types.PackageReference) error

	// PackageReferencePager returns a ReferencePager for dumps that belong to a remote repository (distinct from the given repository id)
	// and reference the package with the given scheme, name, and version. All resulting dumps are visible at the tip of their repository's
	// default branch.
	PackageReferencePager(ctx context.Context, scheme, name, version string, repositoryID, limit int) (int, ReferencePager, error)

	// HasRepository determines if there is LSIF data for the given repository.
	HasRepository(ctx context.Context, repositoryID int) (bool, error)

	// HasCommit determines if the given commit is known for the given repository.
	HasCommit(ctx context.Context, repositoryID int, commit string) (bool, error)

	// MarkRepositoryAsDirty marks the given repository's commit graph as out of date.
	MarkRepositoryAsDirty(ctx context.Context, repositoryID int) error

	// DirtyRepositories returns a map from repository identifiers to a dirty token for each repository whose commit
	// graph is out of date. This token should be passed to CalculateVisibleUploads in order to unmark the repository.
	DirtyRepositories(ctx context.Context) (map[int]int, error)

	// CalculateVisibleUploads uses the given commit graph and the tip commit of the default branch to determine the set
	// of LSIF uploads that are visible for each commit, and the set of uploads which are visible at the tip. The decorated
	// commit graph is serialized to Postgres for use by find closest dumps queries.
	//
	// If dirtyToken is supplied, the repository will be unmarked when the supplied token does matches the most recent
	// token stored in the database, the flag will not be cleared as another request for update has come in since this
	// token has been read.
	CalculateVisibleUploads(ctx context.Context, repositoryID int, graph map[string][]string, tipCommit string, dirtyToken int) error

	// IndexableRepositories returns the identifiers of all indexable repositories.
	IndexableRepositories(ctx context.Context, opts IndexableRepositoryQueryOptions) ([]IndexableRepository, error)

	// UpdateIndexableRepository updates the metadata for an indexable repository. If the repository is not
	// already marked as indexable, a new record will be created.
	UpdateIndexableRepository(ctx context.Context, indexableRepository UpdateableIndexableRepository, now time.Time) error

	// ResetIndexableRepositories zeroes the event counts for indexable repositories that have not been updated
	// since lastUpdatedBefore.
	ResetIndexableRepositories(ctx context.Context, lastUpdatedBefore time.Time) error

	// GetIndexByID returns an index by its identifier and boolean flag indicating its existence.
	GetIndexByID(ctx context.Context, id int) (Index, bool, error)

	// GetIndexes returns a list of indexes and the total count of records matching the given conditions.
	GetIndexes(ctx context.Context, opts GetIndexesOptions) ([]Index, int, error)

	// IndexQueueSize returns the number of indexes in the queued state.
	IndexQueueSize(ctx context.Context) (int, error)

	// IsQueued returns true if there is an index or an upload for the repository and commit.
	IsQueued(ctx context.Context, repositoryID int, commit string) (bool, error)

	// InsertIndex inserts a new index and returns its identifier.
	InsertIndex(ctx context.Context, index Index) (int, error)

	// MarkIndexComplete updates the state of the index to complete.
	MarkIndexComplete(ctx context.Context, id int) (err error)

	// MarkIndexErrored updates the state of the index to errored and updates the failure summary data.
	MarkIndexErrored(ctx context.Context, id int, failureMessage string) error

	// SetIndexLogContents updates the log contents fo the index.
	SetIndexLogContents(ctx context.Context, indexID int, contents string) error

	// DequeueIndex selects the oldest queued index and locks it with a transaction. If there is such an index,
	// the index is returned along with a store instance which wraps the transaction. This transaction must be
	// closed. If there is no such unlocked index, a zero-value index and nil store will be returned along with
	// a false valued flag. This method must not be called from within a transaction.
	DequeueIndex(ctx context.Context) (Index, Store, bool, error)

	// RequeueIndex updates the state of the index to queued and adds a processing delay before the next dequeue attempt.
	RequeueIndex(ctx context.Context, id int, after time.Time) error

	// DeleteIndexByID deletes an index by its identifier.
	DeleteIndexByID(ctx context.Context, id int) (bool, error)

	// DeleteIndexesWithoutRepository deletes indexes associated with repositories that were deleted at least
	// DeletedRepositoryGracePeriod ago. This returns the repository identifier mapped to the number of indexes
	// that were removed for that repository.
	DeleteIndexesWithoutRepository(ctx context.Context, now time.Time) (map[int]int, error)

	// ResetStalledIndexes moves all unlocked index processing for more than `StalledIndexMaxAge` back to the
	// queued state. In order to prevent input that continually crashes indexer instances, indexes that have
	// been reset more than IndexMaxNumResets times will be marked as errored. This method returns a list of
	// updated and errored index identifiers.
	ResetStalledIndexes(ctx context.Context, now time.Time) ([]int, []int, error)

	// RepoUsageStatistics reads recent event log records and returns the number of search-based and precise
	// code intelligence activity within the last week grouped by repository. The resulting slice is ordered
	// by search then precise event counts.
	RepoUsageStatistics(ctx context.Context) ([]RepoUsageStatistics, error)

	// RepoName returns the name for the repo with the given identifier.
	RepoName(ctx context.Context, repositoryID int) (string, error)

	// GetRepositoriesWithIndexConfiguration returns the ids of repositories explicit index configuration.
	GetRepositoriesWithIndexConfiguration(ctx context.Context) ([]int, error)

	// GetIndexConfigurationByRepositoryID returns the index configuration for a repository.
	GetIndexConfigurationByRepositoryID(ctx context.Context, repositoryID int) (IndexConfiguration, bool, error)

	// DeleteUploadsStuckUploading soft deletes any upload record that has been uploading since the given time.
	DeleteUploadsStuckUploading(ctx context.Context, uploadedBefore time.Time) (_ int, err error)
}

type store struct {
	*basestore.Store
}

var _ Store = &store{}

// New creates a new instance of store connected to the given Postgres DSN.
func New(postgresDSN string) (Store, error) {
	base, err := basestore.New(postgresDSN, "codeintel", sql.TxOptions{})
	if err != nil {
		return nil, err
	}

	return &store{Store: base}, nil
}

func NewWithDB(db dbutil.DB) Store {
	return &store{Store: basestore.NewWithDB(db, sql.TxOptions{})}
}

func NewWithHandle(handle *basestore.TransactableHandle) Store {
	return &store{Store: basestore.NewWithHandle(handle)}
}

func (s *store) With(other basestore.ShareableStore) Store {
	return &store{Store: s.Store.With(other)}
}

func (s *store) Transact(ctx context.Context) (Store, error) {
	return s.transact(ctx)
}

func (s *store) transact(ctx context.Context) (*store, error) {
	txBase, err := s.Store.Transact(ctx)
	return &store{Store: txBase}, err
}

// intsToQueries converts a slice of ints into a slice of queries.
func intsToQueries(values []int) []*sqlf.Query {
	var queries []*sqlf.Query
	for _, value := range values {
		queries = append(queries, sqlf.Sprintf("%d", value))
	}

	return queries
}
