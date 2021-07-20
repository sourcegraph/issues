package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/search"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
)

// ChangesetColumns are used by by the changeset related Store methods and by
// workerutil.Worker to load changesets from the database for processing by
// the reconciler.
var ChangesetColumns = []*sqlf.Query{
	sqlf.Sprintf("changesets.id"),
	sqlf.Sprintf("changesets.repo_id"),
	sqlf.Sprintf("changesets.created_at"),
	sqlf.Sprintf("changesets.updated_at"),
	sqlf.Sprintf("changesets.metadata"),
	sqlf.Sprintf("changesets.batch_change_ids"),
	sqlf.Sprintf("changesets.external_id"),
	sqlf.Sprintf("changesets.external_service_type"),
	sqlf.Sprintf("changesets.external_branch"),
	sqlf.Sprintf("changesets.external_deleted_at"),
	sqlf.Sprintf("changesets.external_updated_at"),
	sqlf.Sprintf("changesets.external_state"),
	sqlf.Sprintf("changesets.external_review_state"),
	sqlf.Sprintf("changesets.external_check_state"),
	sqlf.Sprintf("changesets.diff_stat_added"),
	sqlf.Sprintf("changesets.diff_stat_changed"),
	sqlf.Sprintf("changesets.diff_stat_deleted"),
	sqlf.Sprintf("changesets.sync_state"),
	sqlf.Sprintf("changesets.owned_by_batch_change_id"),
	sqlf.Sprintf("changesets.current_spec_id"),
	sqlf.Sprintf("changesets.previous_spec_id"),
	sqlf.Sprintf("changesets.publication_state"),
	sqlf.Sprintf("changesets.ui_publication_state"),
	sqlf.Sprintf("changesets.reconciler_state"),
	sqlf.Sprintf("changesets.failure_message"),
	sqlf.Sprintf("changesets.started_at"),
	sqlf.Sprintf("changesets.finished_at"),
	sqlf.Sprintf("changesets.process_after"),
	sqlf.Sprintf("changesets.num_resets"),
	sqlf.Sprintf("changesets.num_failures"),
	sqlf.Sprintf("changesets.closing"),
	sqlf.Sprintf("changesets.syncer_error"),
}

// changesetInsertColumns is the list of changeset columns that are modified in
// CreateChangeset and UpdateChangeset.
var changesetInsertColumns = []*sqlf.Query{
	sqlf.Sprintf("repo_id"),
	sqlf.Sprintf("created_at"),
	sqlf.Sprintf("updated_at"),
	sqlf.Sprintf("metadata"),
	sqlf.Sprintf("batch_change_ids"),
	sqlf.Sprintf("external_id"),
	sqlf.Sprintf("external_service_type"),
	sqlf.Sprintf("external_branch"),
	sqlf.Sprintf("external_deleted_at"),
	sqlf.Sprintf("external_updated_at"),
	sqlf.Sprintf("external_state"),
	sqlf.Sprintf("external_review_state"),
	sqlf.Sprintf("external_check_state"),
	sqlf.Sprintf("diff_stat_added"),
	sqlf.Sprintf("diff_stat_changed"),
	sqlf.Sprintf("diff_stat_deleted"),
	sqlf.Sprintf("sync_state"),
	sqlf.Sprintf("owned_by_batch_change_id"),
	sqlf.Sprintf("current_spec_id"),
	sqlf.Sprintf("previous_spec_id"),
	sqlf.Sprintf("publication_state"),
	sqlf.Sprintf("ui_publication_state"),
	sqlf.Sprintf("reconciler_state"),
	sqlf.Sprintf("failure_message"),
	sqlf.Sprintf("started_at"),
	sqlf.Sprintf("finished_at"),
	sqlf.Sprintf("process_after"),
	sqlf.Sprintf("num_resets"),
	sqlf.Sprintf("num_failures"),
	sqlf.Sprintf("closing"),
	sqlf.Sprintf("syncer_error"),
	// We additionally store the result of changeset.Title() in a column, so
	// the business logic for determining it is in one place and the field is
	// indexable for searching.
	sqlf.Sprintf("external_title"),
}

// changesetCodeHostStateInsertColumns XX
var changesetCodeHostStateInsertColumns = []*sqlf.Query{
	sqlf.Sprintf("updated_at"),
	sqlf.Sprintf("metadata"),
	sqlf.Sprintf("external_branch"),
	sqlf.Sprintf("external_deleted_at"),
	sqlf.Sprintf("external_updated_at"),
	sqlf.Sprintf("external_state"),
	sqlf.Sprintf("external_review_state"),
	sqlf.Sprintf("external_check_state"),
	sqlf.Sprintf("diff_stat_added"),
	sqlf.Sprintf("diff_stat_changed"),
	sqlf.Sprintf("diff_stat_deleted"),
	sqlf.Sprintf("sync_state"),
	sqlf.Sprintf("syncer_error"),
	// We additionally store the result of changeset.Title() in a column, so
	// the business logic for determining it is in one place and the field is
	// indexable for searching.
	sqlf.Sprintf("external_title"),
}

func (s *Store) changesetWriteQuery(q string, includeID bool, c *btypes.Changeset) (*sqlf.Query, error) {
	metadata, err := jsonbColumn(c.Metadata)
	if err != nil {
		return nil, err
	}

	assocsAsMap := make(map[int64]btypes.BatchChangeAssoc, len(c.BatchChanges))
	for _, assoc := range c.BatchChanges {
		assocsAsMap[assoc.BatchChangeID] = assoc
	}

	batchChanges, err := json.Marshal(assocsAsMap)
	if err != nil {
		return nil, err
	}

	syncState, err := json.Marshal(c.SyncState)
	if err != nil {
		return nil, err
	}

	// Not being able to find a title is fine, we just have a NULL in the database then.
	title, _ := c.Title()

	var uiPublicationState *string
	if state := c.UiPublicationState; state != nil {
		uiPublicationState = nullStringColumn(string(*state))
	}

	vars := []interface{}{
		sqlf.Join(changesetInsertColumns, ", "),
		c.RepoID,
		c.CreatedAt,
		c.UpdatedAt,
		metadata,
		batchChanges,
		nullStringColumn(c.ExternalID),
		c.ExternalServiceType,
		nullStringColumn(c.ExternalBranch),
		nullTimeColumn(c.ExternalDeletedAt),
		nullTimeColumn(c.ExternalUpdatedAt),
		nullStringColumn(string(c.ExternalState)),
		nullStringColumn(string(c.ExternalReviewState)),
		nullStringColumn(string(c.ExternalCheckState)),
		c.DiffStatAdded,
		c.DiffStatChanged,
		c.DiffStatDeleted,
		syncState,
		nullInt64Column(c.OwnedByBatchChangeID),
		nullInt64Column(c.CurrentSpecID),
		nullInt64Column(c.PreviousSpecID),
		c.PublicationState,
		uiPublicationState,
		c.ReconcilerState.ToDB(),
		c.FailureMessage,
		nullTimeColumn(c.StartedAt),
		nullTimeColumn(c.FinishedAt),
		nullTimeColumn(c.ProcessAfter),
		c.NumResets,
		c.NumFailures,
		c.Closing,
		c.SyncErrorMessage,
		nullStringColumn(title),
	}

	if includeID {
		vars = append(vars, c.ID)
	}

	vars = append(vars, sqlf.Join(ChangesetColumns, ", "))

	return sqlf.Sprintf(q, vars...), nil
}

// UpsertChangeset creates or updates the given Changeset.
func (s *Store) UpsertChangeset(ctx context.Context, c *btypes.Changeset) error {
	if c.ID == 0 {
		return s.CreateChangeset(ctx, c)
	}
	return s.UpdateChangeset(ctx, c)
}

// CreateChangeset creates the given Changeset.
func (s *Store) CreateChangeset(ctx context.Context, c *btypes.Changeset) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = s.now()
	}

	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = c.CreatedAt
	}

	q, err := s.changesetWriteQuery(createChangesetQueryFmtstr, false, c)
	if err != nil {
		return err
	}

	return s.query(ctx, q, func(sc scanner) error { return scanChangeset(c, sc) })
}

var createChangesetQueryFmtstr = `
-- source: enterprise/internal/batches/store.go:CreateChangeset
INSERT INTO changesets (%s)
VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
RETURNING %s
`

// DeleteChangeset deletes the Changeset with the given ID.
func (s *Store) DeleteChangeset(ctx context.Context, id int64) error {
	return s.Store.Exec(ctx, sqlf.Sprintf(deleteChangesetQueryFmtstr, id))
}

var deleteChangesetQueryFmtstr = `
DELETE FROM changesets WHERE id = %s
`

// CountChangesetsOpts captures the query options needed for
// counting changesets.
type CountChangesetsOpts struct {
	BatchChangeID        int64
	OnlyArchived         bool
	IncludeArchived      bool
	ExternalStates       []btypes.ChangesetExternalState
	ExternalReviewState  *btypes.ChangesetReviewState
	ExternalCheckState   *btypes.ChangesetCheckState
	ReconcilerStates     []btypes.ReconcilerState
	OwnedByBatchChangeID int64
	PublicationState     *btypes.ChangesetPublicationState
	TextSearch           []search.TextSearchTerm
	EnforceAuthz         bool
	RepoID               api.RepoID
}

// CountChangesets returns the number of changesets in the database.
func (s *Store) CountChangesets(ctx context.Context, opts CountChangesetsOpts) (int, error) {
	authzConds, err := database.AuthzQueryConds(ctx, s.Handle().DB())
	if err != nil {
		return 0, errors.Wrap(err, "CountChangesets generating authz query conds")
	}
	return s.queryCount(ctx, countChangesetsQuery(&opts, authzConds))
}

var countChangesetsQueryFmtstr = `
-- source: enterprise/internal/batches/store.go:CountChangesets
SELECT COUNT(changesets.id)
FROM changesets
INNER JOIN repo ON repo.id = changesets.repo_id
%s -- optional LEFT JOIN to changeset_specs if required
WHERE %s
`

func countChangesetsQuery(opts *CountChangesetsOpts, authzConds *sqlf.Query) *sqlf.Query {
	preds := []*sqlf.Query{
		sqlf.Sprintf("repo.deleted_at IS NULL"),
	}
	if opts.BatchChangeID != 0 {
		batchChangeID := strconv.Itoa(int(opts.BatchChangeID))
		preds = append(preds, sqlf.Sprintf("changesets.batch_change_ids ? %s", batchChangeID))
		if opts.OnlyArchived {
			preds = append(preds, archivedInBatchChange(batchChangeID))
		} else if !opts.IncludeArchived {
			preds = append(preds, sqlf.Sprintf("NOT (%s)", archivedInBatchChange(batchChangeID)))
		}
	}
	if opts.PublicationState != nil {
		preds = append(preds, sqlf.Sprintf("changesets.publication_state = %s", *opts.PublicationState))
	}
	if len(opts.ExternalStates) > 0 {
		states := make([]*sqlf.Query, len(opts.ExternalStates))
		for i, externalState := range opts.ExternalStates {
			states[i] = sqlf.Sprintf("%s", externalState)
		}
		preds = append(preds, sqlf.Sprintf("changesets.external_state IN (%s)", sqlf.Join(states, ",")))
	}
	if opts.ExternalReviewState != nil {
		preds = append(preds, sqlf.Sprintf("changesets.external_review_state = %s", *opts.ExternalReviewState))
	}
	if opts.ExternalCheckState != nil {
		preds = append(preds, sqlf.Sprintf("changesets.external_check_state = %s", *opts.ExternalCheckState))
	}
	if len(opts.ReconcilerStates) != 0 {
		states := make([]*sqlf.Query, len(opts.ReconcilerStates))
		for i, reconcilerState := range opts.ReconcilerStates {
			states[i] = sqlf.Sprintf("%s", reconcilerState.ToDB())
		}
		preds = append(preds, sqlf.Sprintf("changesets.reconciler_state IN (%s)", sqlf.Join(states, ",")))
	}
	if opts.OwnedByBatchChangeID != 0 {
		preds = append(preds, sqlf.Sprintf("changesets.owned_by_batch_change_id = %s", opts.OwnedByBatchChangeID))
	}
	if opts.EnforceAuthz {
		preds = append(preds, authzConds)
	}
	if opts.RepoID != 0 {
		preds = append(preds, sqlf.Sprintf("repo.id = %s", opts.RepoID))
	}

	join := sqlf.Sprintf("")
	if len(opts.TextSearch) != 0 {
		// TextSearch predicates require changeset_specs to be joined into the
		// query as well.
		join = sqlf.Sprintf("LEFT JOIN changeset_specs ON changesets.current_spec_id = changeset_specs.id")

		for _, term := range opts.TextSearch {
			preds = append(preds, textSearchTermToClause(
				term,
				// The COALESCE() is required to handle the actual title on the
				// changeset, if it has been published or if it's tracked.
				sqlf.Sprintf("COALESCE(changesets.external_title, changeset_specs.title)"),
				sqlf.Sprintf("repo.name"),
			))
		}
	}

	return sqlf.Sprintf(countChangesetsQueryFmtstr, join, sqlf.Join(preds, "\n AND "))
}

// GetChangesetByID is a convenience method if only the ID needs to be passed in. It's also used for abstraction in
// the testing package.
func (s *Store) GetChangesetByID(ctx context.Context, id int64) (*btypes.Changeset, error) {
	return s.GetChangeset(ctx, GetChangesetOpts{ID: id})
}

// GetChangesetOpts captures the query options needed for getting a Changeset
type GetChangesetOpts struct {
	ID                  int64
	RepoID              api.RepoID
	ExternalID          string
	ExternalServiceType string
	ExternalBranch      string
	ReconcilerState     btypes.ReconcilerState
	PublicationState    btypes.ChangesetPublicationState
}

// GetChangeset gets a changeset matching the given options.
func (s *Store) GetChangeset(ctx context.Context, opts GetChangesetOpts) (*btypes.Changeset, error) {
	q := getChangesetQuery(&opts)

	var c btypes.Changeset
	err := s.query(ctx, q, func(sc scanner) error { return scanChangeset(&c, sc) })
	if err != nil {
		return nil, err
	}

	if c.ID == 0 {
		return nil, ErrNoResults
	}

	return &c, nil
}

var getChangesetsQueryFmtstr = `
-- source: enterprise/internal/batches/store.go:GetChangeset
SELECT %s FROM changesets
INNER JOIN repo ON repo.id = changesets.repo_id
WHERE %s
LIMIT 1
`

func getChangesetQuery(opts *GetChangesetOpts) *sqlf.Query {
	preds := []*sqlf.Query{
		sqlf.Sprintf("repo.deleted_at IS NULL"),
	}
	if opts.ID != 0 {
		preds = append(preds, sqlf.Sprintf("changesets.id = %s", opts.ID))
	}

	if opts.RepoID != 0 {
		preds = append(preds, sqlf.Sprintf("changesets.repo_id = %s", opts.RepoID))
	}

	if opts.ExternalID != "" && opts.ExternalServiceType != "" {
		preds = append(preds,
			sqlf.Sprintf("changesets.external_id = %s", opts.ExternalID),
			sqlf.Sprintf("changesets.external_service_type = %s", opts.ExternalServiceType),
		)
	}
	if opts.ExternalBranch != "" {
		preds = append(preds, sqlf.Sprintf("changesets.external_branch = %s", opts.ExternalBranch))
	}
	if opts.ReconcilerState != "" {
		preds = append(preds, sqlf.Sprintf("changesets.reconciler_state = %s", opts.ReconcilerState.ToDB()))
	}
	if opts.PublicationState != "" {
		preds = append(preds, sqlf.Sprintf("changesets.publication_state = %s", opts.PublicationState))
	}

	return sqlf.Sprintf(
		getChangesetsQueryFmtstr,
		sqlf.Join(ChangesetColumns, ", "),
		sqlf.Join(preds, "\n AND "),
	)
}

type ListChangesetSyncDataOpts struct {
	// Return only the supplied changesets. If empty, all changesets are returned
	ChangesetIDs []int64

	ExternalServiceID string
}

// ListChangesetSyncData returns sync data on all non-externally-deleted changesets
// that are part of at least one open batch change.
func (s *Store) ListChangesetSyncData(ctx context.Context, opts ListChangesetSyncDataOpts) ([]*btypes.ChangesetSyncData, error) {
	q := listChangesetSyncDataQuery(opts)
	results := make([]*btypes.ChangesetSyncData, 0)
	err := s.query(ctx, q, func(sc scanner) (err error) {
		var h btypes.ChangesetSyncData
		if err := scanChangesetSyncData(&h, sc); err != nil {
			return err
		}
		results = append(results, &h)
		return err
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func scanChangesetSyncData(h *btypes.ChangesetSyncData, s scanner) error {
	return s.Scan(
		&h.ChangesetID,
		&h.UpdatedAt,
		&dbutil.NullTime{Time: &h.LatestEvent},
		&dbutil.NullTime{Time: &h.ExternalUpdatedAt},
		&h.RepoExternalServiceID,
	)
}

const listChangesetSyncDataQueryFmtstr = `
-- source: enterprise/internal/batches/store_changesets.go:ListChangesetSyncData
SELECT changesets.id,
	changesets.updated_at,
	max(ce.updated_at) AS latest_event,
	changesets.external_updated_at,
	r.external_service_id
FROM changesets
LEFT JOIN changeset_events ce ON changesets.id = ce.changeset_id
JOIN batch_changes ON changesets.batch_change_ids ? batch_changes.id::TEXT
JOIN repo r ON changesets.repo_id = r.id
WHERE %s
GROUP BY changesets.id, r.id
ORDER BY changesets.id ASC
`

func listChangesetSyncDataQuery(opts ListChangesetSyncDataOpts) *sqlf.Query {
	preds := []*sqlf.Query{
		sqlf.Sprintf("batch_changes.closed_at IS NULL"),
		sqlf.Sprintf("r.deleted_at IS NULL"),
		sqlf.Sprintf("changesets.publication_state = %s", btypes.ChangesetPublicationStatePublished),
		sqlf.Sprintf("changesets.reconciler_state = %s", btypes.ReconcilerStateCompleted.ToDB()),
	}
	if len(opts.ChangesetIDs) > 0 {
		ids := make([]*sqlf.Query, 0, len(opts.ChangesetIDs))
		for _, id := range opts.ChangesetIDs {
			if id != 0 {
				ids = append(ids, sqlf.Sprintf("%d", id))
			}
		}
		preds = append(preds, sqlf.Sprintf("changesets.id IN (%s)", sqlf.Join(ids, ",")))
	}

	if opts.ExternalServiceID != "" {
		preds = append(preds, sqlf.Sprintf("r.external_service_id = %s", opts.ExternalServiceID))
	}

	return sqlf.Sprintf(listChangesetSyncDataQueryFmtstr, sqlf.Join(preds, "\n AND"))
}

// ListChangesetsOpts captures the query options needed for listing changesets.
//
// Note that TextSearch is potentially expensive, and should only be specified
// in conjunction with at least one other option (most likely, BatchChangeID).
type ListChangesetsOpts struct {
	LimitOpts
	Cursor               int64
	BatchChangeID        int64
	OnlyArchived         bool
	IncludeArchived      bool
	IDs                  []int64
	PublicationState     *btypes.ChangesetPublicationState
	ReconcilerStates     []btypes.ReconcilerState
	ExternalStates       []btypes.ChangesetExternalState
	ExternalReviewState  *btypes.ChangesetReviewState
	ExternalCheckState   *btypes.ChangesetCheckState
	OwnedByBatchChangeID int64
	TextSearch           []search.TextSearchTerm
	EnforceAuthz         bool
	RepoID               api.RepoID
}

// ListChangesets lists Changesets with the given filters.
func (s *Store) ListChangesets(ctx context.Context, opts ListChangesetsOpts) (cs btypes.Changesets, next int64, err error) {
	authzConds, err := database.AuthzQueryConds(ctx, s.Handle().DB())
	if err != nil {
		return nil, 0, errors.Wrap(err, "ListChangesets generating authz query conds")
	}
	q := listChangesetsQuery(&opts, authzConds)

	cs = make([]*btypes.Changeset, 0, opts.DBLimit())
	err = s.query(ctx, q, func(sc scanner) (err error) {
		var c btypes.Changeset
		if err = scanChangeset(&c, sc); err != nil {
			return err
		}
		cs = append(cs, &c)
		return nil
	})

	if opts.Limit != 0 && len(cs) == opts.DBLimit() {
		next = cs[len(cs)-1].ID
		cs = cs[:len(cs)-1]
	}

	return cs, next, err
}

var listChangesetsQueryFmtstr = `
-- source: enterprise/internal/batches/store.go:ListChangesets
SELECT %s FROM changesets
INNER JOIN repo ON repo.id = changesets.repo_id
%s -- optional LEFT JOIN to changeset_specs if required
WHERE %s
ORDER BY id ASC
`

func listChangesetsQuery(opts *ListChangesetsOpts, authzConds *sqlf.Query) *sqlf.Query {
	preds := []*sqlf.Query{
		sqlf.Sprintf("changesets.id >= %s", opts.Cursor),
		sqlf.Sprintf("repo.deleted_at IS NULL"),
	}

	if opts.BatchChangeID != 0 {
		batchChangeID := strconv.Itoa(int(opts.BatchChangeID))
		preds = append(preds, sqlf.Sprintf("changesets.batch_change_ids ? %s", batchChangeID))

		if opts.OnlyArchived {
			preds = append(preds, archivedInBatchChange(batchChangeID))
		} else if !opts.IncludeArchived {
			preds = append(preds, sqlf.Sprintf("NOT (%s)", archivedInBatchChange(batchChangeID)))
		}
	}

	if len(opts.IDs) > 0 {
		ids := make([]*sqlf.Query, 0, len(opts.IDs))
		for _, id := range opts.IDs {
			if id != 0 {
				ids = append(ids, sqlf.Sprintf("%d", id))
			}
		}
		preds = append(preds, sqlf.Sprintf("changesets.id IN (%s)", sqlf.Join(ids, ",")))
	}

	if opts.PublicationState != nil {
		preds = append(preds, sqlf.Sprintf("changesets.publication_state = %s", *opts.PublicationState))
	}
	if len(opts.ReconcilerStates) != 0 {
		states := make([]*sqlf.Query, len(opts.ReconcilerStates))
		for i, reconcilerState := range opts.ReconcilerStates {
			states[i] = sqlf.Sprintf("%s", reconcilerState.ToDB())
		}
		preds = append(preds, sqlf.Sprintf("changesets.reconciler_state IN (%s)", sqlf.Join(states, ",")))
	}
	if len(opts.ExternalStates) > 0 {
		states := make([]*sqlf.Query, len(opts.ExternalStates))
		for i, externalState := range opts.ExternalStates {
			states[i] = sqlf.Sprintf("%s", externalState)
		}
		preds = append(preds, sqlf.Sprintf("changesets.external_state IN (%s)", sqlf.Join(states, ",")))
	}
	if opts.ExternalReviewState != nil {
		preds = append(preds, sqlf.Sprintf("changesets.external_review_state = %s", *opts.ExternalReviewState))
	}
	if opts.ExternalCheckState != nil {
		preds = append(preds, sqlf.Sprintf("changesets.external_check_state = %s", *opts.ExternalCheckState))
	}
	if opts.OwnedByBatchChangeID != 0 {
		preds = append(preds, sqlf.Sprintf("changesets.owned_by_batch_change_id = %s", opts.OwnedByBatchChangeID))
	}
	if opts.EnforceAuthz {
		preds = append(preds, authzConds)
	}
	if opts.RepoID != 0 {
		preds = append(preds, sqlf.Sprintf("repo.id = %s", opts.RepoID))
	}

	join := sqlf.Sprintf("")
	if len(opts.TextSearch) != 0 {
		// TextSearch predicates require changeset_specs to be joined into the
		// query as well.
		join = sqlf.Sprintf("LEFT JOIN changeset_specs ON changesets.current_spec_id = changeset_specs.id")

		for _, term := range opts.TextSearch {
			preds = append(preds, textSearchTermToClause(
				term,
				// The COALESCE() is required to handle the actual title on the
				// changeset, if it has been published or if it's tracked.
				sqlf.Sprintf("COALESCE(changesets.external_title, changeset_specs.title)"),
				sqlf.Sprintf("repo.name"),
			))
		}
	}

	return sqlf.Sprintf(
		listChangesetsQueryFmtstr+opts.LimitOpts.ToDB(),
		sqlf.Join(ChangesetColumns, ", "),
		join,
		sqlf.Join(preds, "\n AND "),
	)
}

// EnqueueChangeset enqueues the given changeset by resetting all
// worker-related columns and setting its reconciler_state column to the
// `resetState` argument but *only if* the `currentState` matches its current
// `reconciler_state`.
func (s *Store) EnqueueChangeset(ctx context.Context, cs *btypes.Changeset, resetState, currentState btypes.ReconcilerState) error {
	_, ok, err := basestore.ScanFirstInt(s.Store.Query(
		ctx,
		s.enqueueChangesetQuery(cs, resetState, currentState),
	))
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("cannot re-enqueue changeset not in failed state")
	}

	return nil
}

var enqueueChangesetQueryFmtstr = `
-- source: enterprise/internal/batches/store/changesets.go:EnqueueChangeset
UPDATE changesets
SET
	reconciler_state = %s,
	num_resets = 0,
	num_failures = 0,
	failure_message = NULL,
	syncer_error = NULL,
	updated_at = %s
WHERE
	%s
RETURNING
	changesets.id
`

func (s *Store) enqueueChangesetQuery(cs *btypes.Changeset, resetState, currentState btypes.ReconcilerState) *sqlf.Query {
	preds := []*sqlf.Query{
		sqlf.Sprintf("id = %s", cs.ID),
	}

	if currentState != "" {
		preds = append(preds, sqlf.Sprintf("reconciler_state = %s", currentState.ToDB()))
	}

	return sqlf.Sprintf(
		enqueueChangesetQueryFmtstr,
		resetState.ToDB(),
		s.now(),
		sqlf.Join(preds, "AND"),
	)
}

// UpdateChangeset updates the given Changeset.
func (s *Store) UpdateChangeset(ctx context.Context, cs *btypes.Changeset) error {
	cs.UpdatedAt = s.now()

	q, err := s.changesetWriteQuery(updateChangesetQueryFmtstr, true, cs)
	if err != nil {
		return err
	}

	return s.query(ctx, q, func(sc scanner) (err error) {
		return scanChangeset(cs, sc)
	})
}

var updateChangesetQueryFmtstr = `
-- source: enterprise/internal/batches/store_changesets.go:UpdateChangeset
UPDATE changesets
SET (%s) = (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
WHERE id = %s
RETURNING
  %s
`

// UpdateChangesetCodeHostState updates only the columns of the given Changeset
// that relate to the state of the changeset on the code host, e.g.
// external_branch, external_state, etc.
func (s *Store) UpdateChangesetCodeHostState(ctx context.Context, cs *btypes.Changeset) error {
	cs.UpdatedAt = s.now()

	q, err := updateChangesetCodeHostStateQuery(cs)
	if err != nil {
		return err
	}

	return s.query(ctx, q, func(sc scanner) (err error) {
		return scanChangeset(cs, sc)
	})
}

func updateChangesetCodeHostStateQuery(c *btypes.Changeset) (*sqlf.Query, error) {
	metadata, err := jsonbColumn(c.Metadata)
	if err != nil {
		return nil, err
	}

	syncState, err := json.Marshal(c.SyncState)
	if err != nil {
		return nil, err
	}

	// Not being able to find a title is fine, we just have a NULL in the database then.
	title, _ := c.Title()

	vars := []interface{}{
		sqlf.Join(changesetCodeHostStateInsertColumns, ", "),
		c.UpdatedAt,
		metadata,
		nullStringColumn(c.ExternalBranch),
		nullTimeColumn(c.ExternalDeletedAt),
		nullTimeColumn(c.ExternalUpdatedAt),
		nullStringColumn(string(c.ExternalState)),
		nullStringColumn(string(c.ExternalReviewState)),
		nullStringColumn(string(c.ExternalCheckState)),
		c.DiffStatAdded,
		c.DiffStatChanged,
		c.DiffStatDeleted,
		syncState,
		c.SyncErrorMessage,
		nullStringColumn(title),
		c.ID,
		sqlf.Join(ChangesetColumns, ", "),
	}

	return sqlf.Sprintf(updateChangesetCodeHostStateQueryFmtstr, vars...), nil
}

var updateChangesetCodeHostStateQueryFmtstr = `
-- source: enterprise/internal/batches/store/changesets.go:UpdateChangesetCodeHostState
UPDATE changesets
SET (%s) = (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
WHERE id = %s
RETURNING
  %s
`

// GetChangesetExternalIDs allows us to find the external ids for pull requests based on
// a slice of head refs. We need this in order to match incoming webhooks to pull requests as
// the only information they provide is the remote branch
func (s *Store) GetChangesetExternalIDs(ctx context.Context, spec api.ExternalRepoSpec, refs []string) ([]string, error) {
	queryFmtString := `
	SELECT cs.external_id FROM changesets cs
	JOIN repo r ON cs.repo_id = r.id
	WHERE cs.external_service_type = %s
	AND cs.external_branch IN (%s)
	AND r.external_id = %s
	AND r.external_service_type = %s
	AND r.external_service_id = %s
	AND r.deleted_at IS NULL
	ORDER BY cs.id ASC;
	`

	inClause := make([]*sqlf.Query, 0, len(refs))
	for _, ref := range refs {
		if ref == "" {
			continue
		}
		inClause = append(inClause, sqlf.Sprintf("%s", ref))
	}

	q := sqlf.Sprintf(queryFmtString, spec.ServiceType, sqlf.Join(inClause, ","), spec.ID, spec.ServiceType, spec.ServiceID)
	return basestore.ScanStrings(s.Store.Query(ctx, q))
}

// CanceledChangesetFailureMessage is set on changesets as the FailureMessage
// by CancelQueuedBatchChangeChangesets which is called at the beginning of
// ApplyBatchChange to stop enqueued changesets being processed while we're
// applying the new batch spec.
var CanceledChangesetFailureMessage = "Canceled"

// CancelQueuedBatchChangeChangesets cancels all scheduled, queued, or errored
// changesets that are owned by the given batch change. It blocks until all
// currently processing changesets have finished executing.
func (s *Store) CancelQueuedBatchChangeChangesets(ctx context.Context, batchChangeID int64) error {
	// Just for safety, so we don't end up with stray cancel requests bombarding
	// the DB with 10 requests a second forever:
	ctx, cancel := context.WithDeadline(ctx, s.now().Add(2*time.Minute))
	defer cancel()

	for {
		// Note that we don't cancel queued "syncing" changesets, since their
		// owned_by_batch_change_id is not set. That's on purpose. It's okay if they're
		// being processed after this, since they only pull data and not create
		// changesets on the code hosts.
		q := sqlf.Sprintf(
			cancelQueuedBatchChangeChangesetsFmtstr,
			batchChangeID,
			btypes.ReconcilerStateScheduled.ToDB(),
			btypes.ReconcilerStateQueued.ToDB(),
			btypes.ReconcilerStateErrored.ToDB(),
			btypes.ReconcilerStateFailed.ToDB(),
			CanceledChangesetFailureMessage,
			batchChangeID,
			btypes.ReconcilerStateProcessing.ToDB(),
		)

		processing, ok, err := basestore.ScanFirstInt(s.Query(ctx, q))
		if err != nil {
			return errors.Wrap(err, "canceling queued batch change changesets failed")
		}
		if !ok || processing == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

const cancelQueuedBatchChangeChangesetsFmtstr = `
-- source: enterprise/internal/batches/store/changesets.go:CancelQueuedBatchChangeChangesets
WITH changeset_ids AS (
  SELECT id FROM changesets
  WHERE
    owned_by_batch_change_id = %s
  AND
    reconciler_state IN (%s, %s, %s)
),
updated_records AS (
	UPDATE
	  changesets
	SET
	  reconciler_state = %s,
	  failure_message = %s
	WHERE id IN (SELECT id FROM changeset_ids)
)
SELECT
	COUNT(id) AS remaining_processing
FROM changesets
WHERE
	owned_by_batch_change_id = %d
	AND
	reconciler_state = %s
`

// EnqueueChangesetsToClose updates all changesets that are owned by the given
// batch change to set their reconciler status to 'queued' and the Closing boolean
// to true.
//
// It does not update the changesets that are fully processed and already
// closed/merged.
//
// This will loop until there are no processing rows anymore, or until 2 minutes
// passed.
func (s *Store) EnqueueChangesetsToClose(ctx context.Context, batchChangeID int64) error {
	// Just for safety, so we don't end up with stray cancel requests bombarding
	// the DB with 10 requests a second forever:
	ctx, cancel := context.WithDeadline(ctx, s.now().Add(2*time.Minute))
	defer cancel()

	for {
		q := sqlf.Sprintf(
			enqueueChangesetsToCloseFmtstr,
			batchChangeID,
			btypes.ChangesetPublicationStatePublished,
			btypes.ReconcilerStateCompleted.ToDB(),
			btypes.ChangesetExternalStateClosed,
			btypes.ChangesetExternalStateMerged,
			btypes.ReconcilerStateQueued.ToDB(),
			btypes.ReconcilerStateProcessing.ToDB(),
			btypes.ReconcilerStateProcessing.ToDB(),
		)
		processing, ok, err := basestore.ScanFirstInt(s.Query(ctx, q))
		if err != nil {
			return err
		}
		if !ok || processing == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

const enqueueChangesetsToCloseFmtstr = `
-- source: enterprise/internal/batches/store_changesets.go:EnqueueChangesetsToClose
WITH all_matching AS (
	SELECT
		id, reconciler_state
	FROM
		changesets
	WHERE
		owned_by_batch_change_id = %d
		AND
		publication_state = %s
		AND
		NOT (
			reconciler_state = %s
			AND
			(external_state = %s OR external_state = %s)
		)
),
updated_records AS (
	UPDATE
		changesets
	SET
		reconciler_state = %s,
		failure_message = NULL,
		num_resets = 0,
		num_failures = 0,
		closing = TRUE
	WHERE
		changesets.id IN (SELECT id FROM all_matching WHERE NOT all_matching.reconciler_state = %s)
)
SELECT COUNT(id) FROM all_matching WHERE all_matching.reconciler_state = %s
`

func ScanFirstChangeset(rows *sql.Rows, err error) (*btypes.Changeset, bool, error) {
	changesets, err := scanChangesets(rows, err)
	if err != nil || len(changesets) == 0 {
		return &btypes.Changeset{}, false, err
	}
	return changesets[0], true, nil
}

func scanChangesets(rows *sql.Rows, queryErr error) ([]*btypes.Changeset, error) {
	if queryErr != nil {
		return nil, queryErr
	}

	var cs []*btypes.Changeset

	return cs, scanAll(rows, func(sc scanner) (err error) {
		var c btypes.Changeset
		if err = scanChangeset(&c, sc); err != nil {
			return err
		}
		cs = append(cs, &c)
		return nil
	})
}

// jsonBatchChangeChangesetSet represents a "join table" set as a JSONB object
// where the keys are the ids and the values are json objects holding the properties.
// It implements the sql.Scanner interface so it can be used as a scan destination,
// similar to sql.NullString.
type jsonBatchChangeChangesetSet struct {
	Assocs *[]btypes.BatchChangeAssoc
}

// Scan implements the Scanner interface.
func (n *jsonBatchChangeChangesetSet) Scan(value interface{}) error {
	m := make(map[int64]btypes.BatchChangeAssoc)

	switch value := value.(type) {
	case nil:
	case []byte:
		if err := json.Unmarshal(value, &m); err != nil {
			return err
		}
	default:
		return errors.Errorf("value is not []byte: %T", value)
	}

	if *n.Assocs == nil {
		*n.Assocs = make([]btypes.BatchChangeAssoc, 0, len(m))
	} else {
		*n.Assocs = (*n.Assocs)[:0]
	}

	for id, assoc := range m {
		assoc.BatchChangeID = id
		*n.Assocs = append(*n.Assocs, assoc)
	}

	return nil
}

// Value implements the driver Valuer interface.
func (n jsonBatchChangeChangesetSet) Value() (driver.Value, error) {
	if n.Assocs == nil {
		return nil, nil
	}
	return *n.Assocs, nil
}

func scanChangeset(t *btypes.Changeset, s scanner) error {
	var metadata, syncState json.RawMessage

	var (
		externalState       string
		externalReviewState string
		externalCheckState  string
		failureMessage      string
		syncErrorMessage    string
		reconcilerState     string
	)
	err := s.Scan(
		&t.ID,
		&t.RepoID,
		&t.CreatedAt,
		&t.UpdatedAt,
		&metadata,
		&jsonBatchChangeChangesetSet{Assocs: &t.BatchChanges},
		&dbutil.NullString{S: &t.ExternalID},
		&t.ExternalServiceType,
		&dbutil.NullString{S: &t.ExternalBranch},
		&dbutil.NullTime{Time: &t.ExternalDeletedAt},
		&dbutil.NullTime{Time: &t.ExternalUpdatedAt},
		&dbutil.NullString{S: &externalState},
		&dbutil.NullString{S: &externalReviewState},
		&dbutil.NullString{S: &externalCheckState},
		&t.DiffStatAdded,
		&t.DiffStatChanged,
		&t.DiffStatDeleted,
		&syncState,
		&dbutil.NullInt64{N: &t.OwnedByBatchChangeID},
		&dbutil.NullInt64{N: &t.CurrentSpecID},
		&dbutil.NullInt64{N: &t.PreviousSpecID},
		&t.PublicationState,
		&t.UiPublicationState,
		&reconcilerState,
		&dbutil.NullString{S: &failureMessage},
		&dbutil.NullTime{Time: &t.StartedAt},
		&dbutil.NullTime{Time: &t.FinishedAt},
		&dbutil.NullTime{Time: &t.ProcessAfter},
		&t.NumResets,
		&t.NumFailures,
		&t.Closing,
		&dbutil.NullString{S: &syncErrorMessage},
	)
	if err != nil {
		return errors.Wrap(err, "scanning changeset")
	}

	t.ExternalState = btypes.ChangesetExternalState(externalState)
	t.ExternalReviewState = btypes.ChangesetReviewState(externalReviewState)
	t.ExternalCheckState = btypes.ChangesetCheckState(externalCheckState)
	if failureMessage != "" {
		t.FailureMessage = &failureMessage
	}
	if syncErrorMessage != "" {
		t.SyncErrorMessage = &syncErrorMessage
	}
	t.ReconcilerState = btypes.ReconcilerState(strings.ToUpper(reconcilerState))

	switch t.ExternalServiceType {
	case extsvc.TypeGitHub:
		t.Metadata = new(github.PullRequest)
	case extsvc.TypeBitbucketServer:
		t.Metadata = new(bitbucketserver.PullRequest)
	case extsvc.TypeGitLab:
		t.Metadata = new(gitlab.MergeRequest)
	default:
		return errors.New("unknown external service type")
	}

	if err = json.Unmarshal(metadata, t.Metadata); err != nil {
		return errors.Wrapf(err, "scanChangeset: failed to unmarshal %q metadata", t.ExternalServiceType)
	}
	if err = json.Unmarshal(syncState, &t.SyncState); err != nil {
		return errors.Wrapf(err, "scanChangeset: failed to unmarshal sync state: %s", syncState)
	}

	return nil
}

// GetChangesetsStats returns statistics on all the changesets associated to the given batch change,
// or all changesets across the instance.
func (s *Store) GetChangesetsStats(ctx context.Context, batchChangeID int64) (stats btypes.ChangesetsStats, err error) {
	q := getChangesetsStatsQuery(batchChangeID)
	err = s.query(ctx, q, func(sc scanner) error {
		if err := sc.Scan(
			&stats.Total,
			&stats.Retrying,
			&stats.Failed,
			&stats.Scheduled,
			&stats.Processing,
			&stats.Unpublished,
			&stats.Closed,
			&stats.Draft,
			&stats.Merged,
			&stats.Open,
			&stats.Deleted,
			&stats.Archived,
		); err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return stats, err
	}
	return stats, nil
}

const getChangesetStatsFmtstr = `
-- source: enterprise/internal/batches/store_changesets.go:GetChangesetsStats
SELECT
	COUNT(*) AS total,
	COUNT(*) FILTER (WHERE changesets.reconciler_state = 'errored') AS retrying,
	COUNT(*) FILTER (WHERE changesets.reconciler_state = 'failed') AS failed,
	COUNT(*) FILTER (WHERE changesets.reconciler_state = 'scheduled') AS scheduled,
	COUNT(*) FILTER (WHERE changesets.reconciler_state NOT IN ('failed', 'errored', 'completed', 'scheduled')) AS processing,
	COUNT(*) FILTER (WHERE changesets.publication_state = 'UNPUBLISHED' AND changesets.reconciler_state = 'completed') AS unpublished,
	COUNT(*) FILTER (WHERE %s AND changesets.external_state = 'CLOSED'  AND NOT %s) AS closed,
	COUNT(*) FILTER (WHERE %s AND changesets.external_state = 'DRAFT'   AND NOT %s) AS draft,
	COUNT(*) FILTER (WHERE %s AND changesets.external_state = 'MERGED'  AND NOT %s) AS merged,
	COUNT(*) FILTER (WHERE %s AND changesets.external_state = 'OPEN'    AND NOT %s) AS open,
	COUNT(*) FILTER (WHERE %s AND changesets.external_state = 'DELETED' AND NOT %s) AS deleted,
	COUNT(*) FILTER (WHERE %s)                                                      AS archived
FROM changesets
INNER JOIN repo on repo.id = changesets.repo_id
WHERE
	%s
`

// GetRepoChangesetsStats returns statistics on all the changesets associated to the given repo.
func (s *Store) GetRepoChangesetsStats(ctx context.Context, repoID api.RepoID) (*btypes.RepoChangesetsStats, error) {
	authzConds, err := database.AuthzQueryConds(ctx, s.Handle().DB())
	if err != nil {
		return nil, errors.Wrap(err, "GetRepoChangesetsStats generating authz query conds")
	}
	q := getRepoChangesetsStatsQuery(int64(repoID), authzConds)

	var stats btypes.RepoChangesetsStats
	err = s.query(ctx, q, func(sc scanner) error {
		if err := sc.Scan(
			&stats.Total,
			&stats.Unpublished,
			&stats.Draft,
			&stats.Closed,
			&stats.Merged,
			&stats.Open,
		); err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return &stats, err
	}
	return &stats, nil
}

func (s *Store) EnqueueNextScheduledChangeset(ctx context.Context) (*btypes.Changeset, error) {
	q := sqlf.Sprintf(
		enqueueNextScheduledChangesetFmtstr,
		btypes.ReconcilerStateScheduled.ToDB(),
		btypes.ReconcilerStateQueued.ToDB(),
		sqlf.Join(ChangesetColumns, ","),
	)

	var c btypes.Changeset
	err := s.query(ctx, q, func(sc scanner) error {
		return scanChangeset(&c, sc)
	})
	if err != nil {
		return nil, err
	}

	if c.ID == 0 {
		return nil, ErrNoResults
	}

	return &c, nil
}

const enqueueNextScheduledChangesetFmtstr = `
-- source: enterprise/internal/batches/store/changesets.go:EnqueueNextScheduledChangeset
WITH c AS (
	SELECT *
	FROM changesets
	WHERE reconciler_state = %s
	ORDER BY updated_at ASC
	LIMIT 1
)
UPDATE changesets
SET reconciler_state = %s
FROM c
WHERE c.id = changesets.id
RETURNING %s
`

func (s *Store) GetChangesetPlaceInSchedulerQueue(ctx context.Context, id int64) (int, error) {
	q := sqlf.Sprintf(
		getChangesetPlaceInSchedulerQueueFmtstr,
		btypes.ReconcilerStateScheduled.ToDB(),
		id,
	)

	row := s.QueryRow(ctx, q)
	var place int
	if err := row.Scan(&place); err == sql.ErrNoRows {
		return 0, ErrNoResults
	} else if err != nil {
		return 0, err
	}

	// PostgreSQL returns 1-indexed row numbers, but we want 0-indexed places
	// when calculating schedules.
	return place - 1, nil
}

const getChangesetPlaceInSchedulerQueueFmtstr = `
-- source: enterprise/internal/batches/store/changesets.go:GetChangesetPlaceInSchedulerQueue
SELECT
	row_number
FROM (
	SELECT
		id,
		ROW_NUMBER() OVER (ORDER BY updated_at ASC) AS row_number
	FROM
		changesets
	WHERE
		reconciler_state = %s
	) t
WHERE
	id = %d
`

func archivedInBatchChange(batchChangeID string) *sqlf.Query {
	return sqlf.Sprintf(
		"(COALESCE((batch_change_ids->%s->>'isArchived')::bool, false) OR COALESCE((batch_change_ids->%s->>'archive')::bool, false))",
		batchChangeID,
		batchChangeID,
	)
}

func getChangesetsStatsQuery(batchChangeID int64) *sqlf.Query {
	batchChangeIDStr := strconv.Itoa(int(batchChangeID))

	preds := []*sqlf.Query{
		sqlf.Sprintf("repo.deleted_at IS NULL"),
		sqlf.Sprintf("changesets.batch_change_ids ? %s", batchChangeIDStr),
	}

	publishedAndCompleted := sqlf.Sprintf("changesets.publication_state = 'PUBLISHED' AND changesets.reconciler_state = 'completed'")
	archived := archivedInBatchChange(batchChangeIDStr)

	return sqlf.Sprintf(
		getChangesetStatsFmtstr,
		publishedAndCompleted, archived,
		publishedAndCompleted, archived,
		publishedAndCompleted, archived,
		publishedAndCompleted, archived,
		publishedAndCompleted, archived,
		archived,
		sqlf.Join(preds, " AND "),
	)
}

func getRepoChangesetsStatsQuery(repoID int64, authzConds *sqlf.Query) *sqlf.Query {
	publishedAndCompleted := sqlf.Sprintf("publication_state = 'PUBLISHED' AND reconciler_state = 'completed'")

	return sqlf.Sprintf(
		getRepoChangesetsStatsFmtstr,
		publishedAndCompleted, publishedAndCompleted, publishedAndCompleted, publishedAndCompleted,
		strconv.Itoa(int(repoID)),
		authzConds,
	)
}

const getRepoChangesetsStatsFmtstr = `
-- source: enterprise/internal/batches/store/changesets.go:GetRepoChangesetsStats
SELECT
	COUNT(*) AS total,
	COUNT(*) FILTER (WHERE publication_state = 'UNPUBLISHED'
		AND reconciler_state = 'completed') AS unpublished,
	COUNT(*) FILTER (WHERE %s AND external_state = 'DRAFT') AS draft,
	COUNT(*) FILTER (WHERE %s AND external_state = 'CLOSED') AS closed,
	COUNT(*) FILTER (WHERE %s AND external_state = 'MERGED') AS merged,
	COUNT(*) FILTER (WHERE %s AND external_state = 'OPEN') AS open
FROM (
	SELECT
		changesets.id,
		changesets.publication_state,
		changesets.reconciler_state,
		changesets.external_state
	FROM
		changesets
		INNER JOIN repo ON changesets.repo_id = repo.id
	WHERE
		repo.id = %s
		-- where the changeset is not archived on at least one batch change
		AND jsonb_path_exists (batch_change_ids, '$.* ? ((!exists(@.isArchived) || @.isArchived == false) && (!exists(@.archive) || @.archive == false))')
		-- authz conditions:
		AND %s
) AS fcs;
`
