package campaigns

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/keegancsmith/sqlf"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
)

// changesetColumns are used by by the changeset related Store methods and by
// workerutil.Worker to load changesets from the database for processing by
// the reconciler.
var changesetColumns = []*sqlf.Query{
	sqlf.Sprintf("changesets.id"),
	sqlf.Sprintf("changesets.repo_id"),
	sqlf.Sprintf("changesets.created_at"),
	sqlf.Sprintf("changesets.updated_at"),
	sqlf.Sprintf("changesets.metadata"),
	sqlf.Sprintf("changesets.campaign_ids"),
	sqlf.Sprintf("changesets.external_id"),
	sqlf.Sprintf("changesets.external_service_type"),
	sqlf.Sprintf("changesets.external_branch"),
	sqlf.Sprintf("changesets.external_deleted_at"),
	sqlf.Sprintf("changesets.external_updated_at"),
	sqlf.Sprintf("changesets.external_state"),
	sqlf.Sprintf("changesets.external_review_state"),
	sqlf.Sprintf("changesets.external_check_state"),
	sqlf.Sprintf("changesets.added_to_campaign"),
	sqlf.Sprintf("changesets.diff_stat_added"),
	sqlf.Sprintf("changesets.diff_stat_changed"),
	sqlf.Sprintf("changesets.diff_stat_deleted"),
	sqlf.Sprintf("changesets.sync_state"),
	sqlf.Sprintf("changesets.owned_by_campaign_id"),
	sqlf.Sprintf("changesets.current_spec_id"),
	sqlf.Sprintf("changesets.previous_spec_id"),
	sqlf.Sprintf("changesets.publication_state"),
	sqlf.Sprintf("changesets.reconciler_state"),
	sqlf.Sprintf("changesets.failure_message"),
	sqlf.Sprintf("changesets.started_at"),
	sqlf.Sprintf("changesets.finished_at"),
	sqlf.Sprintf("changesets.process_after"),
	sqlf.Sprintf("changesets.num_resets"),
	sqlf.Sprintf("changesets.num_failures"),
	sqlf.Sprintf("changesets.unsynced"),
	sqlf.Sprintf("changesets.closing"),
}

// changesetInsertColumns is the list of changeset columns that are modified in
// CreateChangeset and UpdateChangeset.
var changesetInsertColumns = []*sqlf.Query{
	sqlf.Sprintf("repo_id"),
	sqlf.Sprintf("created_at"),
	sqlf.Sprintf("updated_at"),
	sqlf.Sprintf("metadata"),
	sqlf.Sprintf("campaign_ids"),
	sqlf.Sprintf("external_id"),
	sqlf.Sprintf("external_service_type"),
	sqlf.Sprintf("external_branch"),
	sqlf.Sprintf("external_deleted_at"),
	sqlf.Sprintf("external_updated_at"),
	sqlf.Sprintf("external_state"),
	sqlf.Sprintf("external_review_state"),
	sqlf.Sprintf("external_check_state"),
	sqlf.Sprintf("added_to_campaign"),
	sqlf.Sprintf("diff_stat_added"),
	sqlf.Sprintf("diff_stat_changed"),
	sqlf.Sprintf("diff_stat_deleted"),
	sqlf.Sprintf("sync_state"),
	sqlf.Sprintf("owned_by_campaign_id"),
	sqlf.Sprintf("current_spec_id"),
	sqlf.Sprintf("previous_spec_id"),
	sqlf.Sprintf("publication_state"),
	sqlf.Sprintf("reconciler_state"),
	sqlf.Sprintf("failure_message"),
	sqlf.Sprintf("started_at"),
	sqlf.Sprintf("finished_at"),
	sqlf.Sprintf("process_after"),
	sqlf.Sprintf("num_resets"),
	sqlf.Sprintf("num_failures"),
	sqlf.Sprintf("unsynced"),
	sqlf.Sprintf("closing"),
}

func (s *Store) changesetWriteQuery(q string, includeID bool, c *campaigns.Changeset) (*sqlf.Query, error) {
	metadata, err := jsonbColumn(c.Metadata)
	if err != nil {
		return nil, err
	}

	campaignIDs, err := jsonSetColumn(c.CampaignIDs)
	if err != nil {
		return nil, err
	}

	syncState, err := json.Marshal(c.SyncState)
	if err != nil {
		return nil, err
	}

	vars := []interface{}{
		sqlf.Join(changesetInsertColumns, ", "),
		c.RepoID,
		c.CreatedAt,
		c.UpdatedAt,
		metadata,
		campaignIDs,
		nullStringColumn(c.ExternalID),
		c.ExternalServiceType,
		nullStringColumn(c.ExternalBranch),
		nullTimeColumn(c.ExternalDeletedAt),
		nullTimeColumn(c.ExternalUpdatedAt),
		nullStringColumn(string(c.ExternalState)),
		nullStringColumn(string(c.ExternalReviewState)),
		nullStringColumn(string(c.ExternalCheckState)),
		c.AddedToCampaign,
		c.DiffStatAdded,
		c.DiffStatChanged,
		c.DiffStatDeleted,
		syncState,
		nullInt64Column(c.OwnedByCampaignID),
		nullInt64Column(c.CurrentSpecID),
		nullInt64Column(c.PreviousSpecID),
		c.PublicationState,
		c.ReconcilerState.ToDB(),
		c.FailureMessage,
		nullTimeColumn(c.StartedAt),
		nullTimeColumn(c.FinishedAt),
		nullTimeColumn(c.ProcessAfter),
		c.NumResets,
		c.NumFailures,
		c.Unsynced,
		c.Closing,
	}

	if includeID {
		vars = append(vars, c.ID)
	}

	vars = append(vars, sqlf.Join(changesetColumns, ", "))

	return sqlf.Sprintf(q, vars...), nil
}

// CreateChangeset creates the given Changeset.
func (s *Store) CreateChangeset(ctx context.Context, c *campaigns.Changeset) error {
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
-- source: enterprise/internal/campaigns/store.go:CreateChangeset
INSERT INTO changesets (%s)
VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
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
	CampaignID          int64
	ExternalState       *campaigns.ChangesetExternalState
	ExternalReviewState *campaigns.ChangesetReviewState
	ExternalCheckState  *campaigns.ChangesetCheckState
	ReconcilerStates    []campaigns.ReconcilerState
	OwnedByCampaignID   int64
}

// CountChangesets returns the number of changesets in the database.
func (s *Store) CountChangesets(ctx context.Context, opts CountChangesetsOpts) (int, error) {
	return s.queryCount(ctx, countChangesetsQuery(&opts))
}

var countChangesetsQueryFmtstr = `
-- source: enterprise/internal/campaigns/store.go:CountChangesets
SELECT COUNT(changesets.id)
FROM changesets
INNER JOIN repo ON repo.id = changesets.repo_id
WHERE %s
`

func countChangesetsQuery(opts *CountChangesetsOpts) *sqlf.Query {
	preds := []*sqlf.Query{
		sqlf.Sprintf("repo.deleted_at IS NULL"),
	}
	if opts.CampaignID != 0 {
		preds = append(preds, sqlf.Sprintf("changesets.campaign_ids ? %s", opts.CampaignID))
	}

	if opts.ExternalState != nil {
		preds = append(preds, sqlf.Sprintf("changesets.external_state = %s", *opts.ExternalState))
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
	if opts.OwnedByCampaignID != 0 {
		preds = append(preds, sqlf.Sprintf("changesets.owned_by_campaign_id = %s", opts.OwnedByCampaignID))
	}

	return sqlf.Sprintf(countChangesetsQueryFmtstr, sqlf.Join(preds, "\n AND "))
}

// GetChangesetOpts captures the query options needed for getting a Changeset
type GetChangesetOpts struct {
	ID                  int64
	RepoID              api.RepoID
	ExternalID          string
	ExternalServiceType string
	ExternalBranch      string
}

// GetChangeset gets a changeset matching the given options.
func (s *Store) GetChangeset(ctx context.Context, opts GetChangesetOpts) (*campaigns.Changeset, error) {
	q := getChangesetQuery(&opts)

	var c campaigns.Changeset
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
-- source: enterprise/internal/campaigns/store.go:GetChangeset
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

	return sqlf.Sprintf(
		getChangesetsQueryFmtstr,
		sqlf.Join(changesetColumns, ", "),
		sqlf.Join(preds, "\n AND "),
	)
}

type ListChangesetSyncDataOpts struct {
	// Return only the supplied changesets. If empty, all changesets are returned
	ChangesetIDs []int64
}

// ListChangesetSyncData returns sync data on all non-externally-deleted changesets
// that are part of at least one open campaign.
func (s *Store) ListChangesetSyncData(ctx context.Context, opts ListChangesetSyncDataOpts) ([]campaigns.ChangesetSyncData, error) {
	q := listChangesetSyncData(opts)
	results := make([]campaigns.ChangesetSyncData, 0)
	err := s.query(ctx, q, func(sc scanner) (err error) {
		var h campaigns.ChangesetSyncData
		if err = scanChangesetSyncData(&h, sc); err != nil {
			return err
		}
		results = append(results, h)
		return err
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func scanChangesetSyncData(h *campaigns.ChangesetSyncData, s scanner) error {
	return s.Scan(
		&h.ChangesetID,
		&h.UpdatedAt,
		&dbutil.NullTime{Time: &h.LatestEvent},
		&dbutil.NullTime{Time: &h.ExternalUpdatedAt},
		&h.RepoExternalServiceID,
	)
}

func listChangesetSyncData(opts ListChangesetSyncDataOpts) *sqlf.Query {
	fmtString := `
 SELECT changesets.id,
        changesets.updated_at,
        max(ce.updated_at) AS latest_event,
        changesets.external_updated_at,
        r.external_service_id
 FROM changesets
 LEFT JOIN changeset_events ce ON changesets.id = ce.changeset_id
 JOIN campaigns ON campaigns.changeset_ids ? changesets.id::TEXT
 JOIN repo r ON changesets.repo_id = r.id
 WHERE %s
 GROUP BY changesets.id, r.id
 ORDER BY changesets.id ASC
`

	preds := []*sqlf.Query{
		sqlf.Sprintf("campaigns.closed_at IS NULL"),
		sqlf.Sprintf("r.deleted_at IS NULL"),
		sqlf.Sprintf("changesets.publication_state = %s", campaigns.ChangesetPublicationStatePublished),
		sqlf.Sprintf("changesets.reconciler_state = %s", campaigns.ReconcilerStateCompleted.ToDB()),
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

	return sqlf.Sprintf(fmtString, sqlf.Join(preds, "\n AND"))
}

// ListChangesetsOpts captures the query options needed for
// listing changesets.
type ListChangesetsOpts struct {
	LimitOpts
	Cursor               int64
	CampaignID           int64
	IDs                  []int64
	WithoutDeleted       bool
	PublicationState     *campaigns.ChangesetPublicationState
	ReconcilerStates     []campaigns.ReconcilerState
	ExternalState        *campaigns.ChangesetExternalState
	ExternalReviewState  *campaigns.ChangesetReviewState
	ExternalCheckState   *campaigns.ChangesetCheckState
	OwnedByCampaignID    int64
	OnlyWithoutDiffStats bool
	OnlySynced           bool
}

// ListChangesets lists Changesets with the given filters.
func (s *Store) ListChangesets(ctx context.Context, opts ListChangesetsOpts) (cs campaigns.Changesets, next int64, err error) {
	q := listChangesetsQuery(&opts)

	cs = make([]*campaigns.Changeset, 0, opts.DBLimit())
	err = s.query(ctx, q, func(sc scanner) (err error) {
		var c campaigns.Changeset
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
-- source: enterprise/internal/campaigns/store.go:ListChangesets
SELECT %s FROM changesets
INNER JOIN repo ON repo.id = changesets.repo_id
WHERE %s
ORDER BY id ASC
`

func listChangesetsQuery(opts *ListChangesetsOpts) *sqlf.Query {
	preds := []*sqlf.Query{
		sqlf.Sprintf("changesets.id >= %s", opts.Cursor),
		sqlf.Sprintf("repo.deleted_at IS NULL"),
	}

	if opts.CampaignID != 0 {
		preds = append(preds, sqlf.Sprintf("changesets.campaign_ids ? %s", opts.CampaignID))
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

	if opts.WithoutDeleted {
		preds = append(preds, sqlf.Sprintf("changesets.external_deleted_at IS NULL"))
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
	if opts.ExternalState != nil {
		preds = append(preds, sqlf.Sprintf("changesets.external_state = %s", *opts.ExternalState))
	}
	if opts.ExternalReviewState != nil {
		preds = append(preds, sqlf.Sprintf("changesets.external_review_state = %s", *opts.ExternalReviewState))
	}
	if opts.ExternalCheckState != nil {
		preds = append(preds, sqlf.Sprintf("changesets.external_check_state = %s", *opts.ExternalCheckState))
	}
	if opts.OwnedByCampaignID != 0 {
		preds = append(preds, sqlf.Sprintf("changesets.owned_by_campaign_id = %s", opts.OwnedByCampaignID))
	}

	if opts.OnlyWithoutDiffStats {
		preds = append(preds, sqlf.Sprintf("(changesets.diff_stat_added IS NULL OR changesets.diff_stat_changed IS NULL OR changesets.diff_stat_deleted IS NULL)"))
	}

	if opts.OnlySynced {
		preds = append(preds, sqlf.Sprintf("changesets.unsynced IS FALSE"))
	}

	return sqlf.Sprintf(
		listChangesetsQueryFmtstr+opts.LimitOpts.ToDB(),
		sqlf.Join(changesetColumns, ", "),
		sqlf.Join(preds, "\n AND "),
	)
}

// ListChangesetsAttachedOrOwnedByCampaign lists Changesets that are either
// attached to the given Campaign or their OwnedByCampaignID points to the
// campaign.
func (s *Store) ListChangesetsAttachedOrOwnedByCampaign(ctx context.Context, campaign int64) (cs campaigns.Changesets, err error) {
	q := sqlf.Sprintf(`
-- source: enterprise/internal/campaigns/store.go:ListChangesetsAttachedOrOwnedByCampaign
SELECT %s FROM changesets
INNER JOIN repo ON repo.id = changesets.repo_id
WHERE
  ((changesets.campaign_ids ? %s) OR changesets.owned_by_campaign_id = %s)
AND
  repo.deleted_at IS NULL
ORDER BY id ASC
`,
		sqlf.Join(changesetColumns, ", "),
		campaign,
		campaign,
	)

	err = s.query(ctx, q, func(sc scanner) (err error) {
		var c campaigns.Changeset
		if err = scanChangeset(&c, sc); err != nil {
			return err
		}
		cs = append(cs, &c)
		return nil
	})

	return cs, err
}

// UpdateChangeset updates the given Changeset.
func (s *Store) UpdateChangeset(ctx context.Context, cs *campaigns.Changeset) error {
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
-- source: enterprise/internal/campaigns/store_changesets.go:UpdateChangeset
UPDATE changesets
SET (%s) = (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
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

// canceledChangesetFailureMessage is set on changesets as the FailureMessage
// by CancelQueuedCampaignChangesets which is called at the beginning of
// ApplyCampaign to stop enqueued changesets being processed while we're
// applying the new campaign spec.
var canceledChangesetFailureMessage = "Canceled"

func (s *Store) CancelQueuedCampaignChangesets(ctx context.Context, campaignID int64) error {
	// Note that we don't cancel queued "syncing" changesets, since their
	// owned_by_campaign_id is not set. That's on purpose. It's okay if they're
	// being processed after this, since they only pull data and not create
	// changesets on the code hosts.
	q := sqlf.Sprintf(
		cancelQueuedCampaignChangesetsFmtstr,
		campaignID,
		reconcilerMaxNumRetries,
		canceledChangesetFailureMessage,
		reconcilerMaxNumRetries,
	)
	return s.Store.Exec(ctx, q)
}

const cancelQueuedCampaignChangesetsFmtstr = `
-- source: enterprise/internal/campaigns/store_changesets.go:CancelQueuedCampaignChangesets
WITH changeset_ids AS (
  SELECT id FROM changesets
  WHERE
    owned_by_campaign_id = %s
  AND
    (reconciler_state = 'queued' OR
	 reconciler_state = 'processing' OR
	 (reconciler_state = 'errored' AND num_failures < %d))
  FOR UPDATE
)
UPDATE
  changesets
SET
  reconciler_state = 'errored',
  failure_message = %s,
  num_failures = %d
WHERE id IN (SELECT id FROM changeset_ids);
`

// EnqueueChangesetsToClose updates all changesets that are owned by the given
// campaign to set their reconciler status to 'queued' and the Closing boolean
// to true.
//
// It does not update the changesets that are fully processed and already
// closed/merged.
//
// This method will *block* if some of the changesets are currently being processed.
func (s *Store) EnqueueChangesetsToClose(ctx context.Context, campaignID int64) error {
	q := sqlf.Sprintf(
		enqueueChangesetsToCloseFmtstr,
		campaignID,
		campaigns.ChangesetExternalStateClosed,
		campaigns.ChangesetExternalStateMerged,
	)
	return s.Store.Exec(ctx, q)
}

const enqueueChangesetsToCloseFmtstr = `
-- source: enterprise/internal/campaigns/store_changesets.go:EnqueueChangesetsToClose
UPDATE
  changesets
SET
  reconciler_state = 'queued',
  failure_message = NULL,
  num_failures = 0,
  closing = TRUE
WHERE
  owned_by_campaign_id = %d
AND
  NOT (
    reconciler_state = 'completed'
	AND
	(external_state = %s OR external_state = %s)
  )
;
`

func scanFirstChangeset(rows *sql.Rows, err error) (*campaigns.Changeset, bool, error) {
	changesets, err := scanChangesets(rows, err)
	if err != nil || len(changesets) == 0 {
		return &campaigns.Changeset{}, false, err
	}
	return changesets[0], true, nil
}

func scanChangesets(rows *sql.Rows, queryErr error) ([]*campaigns.Changeset, error) {
	if queryErr != nil {
		return nil, queryErr
	}

	var cs []*campaigns.Changeset

	return cs, scanAll(rows, func(sc scanner) (err error) {
		var c campaigns.Changeset
		if err = scanChangeset(&c, sc); err != nil {
			return err
		}
		cs = append(cs, &c)
		return nil
	})
}

func scanChangeset(t *campaigns.Changeset, s scanner) error {
	var metadata, syncState json.RawMessage

	var (
		externalState       string
		externalReviewState string
		externalCheckState  string
		failureMessage      string
		reconcilerState     string
	)
	err := s.Scan(
		&t.ID,
		&t.RepoID,
		&t.CreatedAt,
		&t.UpdatedAt,
		&metadata,
		&dbutil.JSONInt64Set{Set: &t.CampaignIDs},
		&dbutil.NullString{S: &t.ExternalID},
		&t.ExternalServiceType,
		&dbutil.NullString{S: &t.ExternalBranch},
		&dbutil.NullTime{Time: &t.ExternalDeletedAt},
		&dbutil.NullTime{Time: &t.ExternalUpdatedAt},
		&dbutil.NullString{S: &externalState},
		&dbutil.NullString{S: &externalReviewState},
		&dbutil.NullString{S: &externalCheckState},
		&t.AddedToCampaign,
		&t.DiffStatAdded,
		&t.DiffStatChanged,
		&t.DiffStatDeleted,
		&syncState,
		&dbutil.NullInt64{N: &t.OwnedByCampaignID},
		&dbutil.NullInt64{N: &t.CurrentSpecID},
		&dbutil.NullInt64{N: &t.PreviousSpecID},
		&t.PublicationState,
		&reconcilerState,
		&dbutil.NullString{S: &failureMessage},
		&dbutil.NullTime{Time: &t.StartedAt},
		&dbutil.NullTime{Time: &t.FinishedAt},
		&dbutil.NullTime{Time: &t.ProcessAfter},
		&t.NumResets,
		&t.NumFailures,
		&t.Unsynced,
		&t.Closing,
	)
	if err != nil {
		return errors.Wrap(err, "scanning changeset")
	}

	t.ExternalState = campaigns.ChangesetExternalState(externalState)
	t.ExternalReviewState = campaigns.ChangesetReviewState(externalReviewState)
	t.ExternalCheckState = campaigns.ChangesetCheckState(externalCheckState)
	if failureMessage != "" {
		t.FailureMessage = &failureMessage
	}
	t.ReconcilerState = campaigns.ReconcilerState(strings.ToUpper(reconcilerState))

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
