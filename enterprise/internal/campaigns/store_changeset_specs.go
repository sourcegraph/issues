package campaigns

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/dineshappavoo/basex"
	"github.com/keegancsmith/sqlf"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

// changesetSpecInsertColumns is the list of changeset_specs columns that are
// modified when inserting or updating a changeset spec.
var changesetSpecInsertColumns = []*sqlf.Query{
	sqlf.Sprintf("rand_id"),
	sqlf.Sprintf("raw_spec"),
	sqlf.Sprintf("spec"),
	sqlf.Sprintf("campaign_spec_id"),
	sqlf.Sprintf("repo_id"),
	sqlf.Sprintf("user_id"),
	sqlf.Sprintf("diff_stat_added"),
	sqlf.Sprintf("diff_stat_changed"),
	sqlf.Sprintf("diff_stat_deleted"),
	sqlf.Sprintf("created_at"),
	sqlf.Sprintf("updated_at"),
}

// changesetSpecColumns are used by the changeset spec related Store methods to
// insert, update and query changeset specs.
var changesetSpecColumns = []*sqlf.Query{
	sqlf.Sprintf("changeset_specs.id"),
	sqlf.Sprintf("changeset_specs.rand_id"),
	sqlf.Sprintf("changeset_specs.raw_spec"),
	sqlf.Sprintf("changeset_specs.spec"),
	sqlf.Sprintf("changeset_specs.campaign_spec_id"),
	sqlf.Sprintf("changeset_specs.repo_id"),
	sqlf.Sprintf("changeset_specs.user_id"),
	sqlf.Sprintf("changeset_specs.diff_stat_added"),
	sqlf.Sprintf("changeset_specs.diff_stat_changed"),
	sqlf.Sprintf("changeset_specs.diff_stat_deleted"),
	sqlf.Sprintf("changeset_specs.created_at"),
	sqlf.Sprintf("changeset_specs.updated_at"),
}

// CreateChangesetSpec creates the given ChangesetSpec.
func (s *Store) CreateChangesetSpec(ctx context.Context, c *campaigns.ChangesetSpec) error {
	q, err := s.createChangesetSpecQuery(c)
	if err != nil {
		return err
	}

	return s.query(ctx, q, func(sc scanner) error { return scanChangesetSpec(c, sc) })
}

var createChangesetSpecQueryFmtstr = `
-- source: enterprise/internal/campaigns/store_changeset_specs.go:CreateChangesetSpec
INSERT INTO changeset_specs (%s)
VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
RETURNING %s`

func (s *Store) createChangesetSpecQuery(c *campaigns.ChangesetSpec) (*sqlf.Query, error) {
	spec, err := jsonbColumn(c.Spec)
	if err != nil {
		return nil, err
	}

	if c.CreatedAt.IsZero() {
		c.CreatedAt = s.now()
	}

	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = c.CreatedAt
	}

	if c.RandID == "" {
		if c.RandID, err = basex.Encode(strconv.Itoa(seededRand.Int())); err != nil {
			return nil, errors.Wrap(err, "creating RandID failed")
		}
	}

	return sqlf.Sprintf(
		createChangesetSpecQueryFmtstr,
		sqlf.Join(changesetSpecInsertColumns, ", "),
		c.RandID,
		c.RawSpec,
		spec,
		nullInt64Column(c.CampaignSpecID),
		c.RepoID,
		nullInt32Column(c.UserID),
		c.DiffStatAdded,
		c.DiffStatChanged,
		c.DiffStatDeleted,
		c.CreatedAt,
		c.UpdatedAt,
		sqlf.Join(changesetSpecColumns, ", "),
	), nil
}

// UpdateChangesetSpec updates the given ChangesetSpec.
func (s *Store) UpdateChangesetSpec(ctx context.Context, c *campaigns.ChangesetSpec) error {
	q, err := s.updateChangesetSpecQuery(c)
	if err != nil {
		return err
	}

	return s.query(ctx, q, func(sc scanner) error {
		return scanChangesetSpec(c, sc)
	})
}

var updateChangesetSpecQueryFmtstr = `
-- source: enterprise/internal/campaigns/store_changeset_specs.go:UpdateChangesetSpec
UPDATE changeset_specs
SET (%s) = (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
WHERE id = %s
RETURNING %s`

func (s *Store) updateChangesetSpecQuery(c *campaigns.ChangesetSpec) (*sqlf.Query, error) {
	spec, err := jsonbColumn(c.Spec)
	if err != nil {
		return nil, err
	}

	c.UpdatedAt = s.now()

	return sqlf.Sprintf(
		updateChangesetSpecQueryFmtstr,
		sqlf.Join(changesetSpecInsertColumns, ", "),
		c.RandID,
		c.RawSpec,
		spec,
		nullInt64Column(c.CampaignSpecID),
		c.RepoID,
		nullInt32Column(c.UserID),
		c.DiffStatAdded,
		c.DiffStatChanged,
		c.DiffStatDeleted,
		c.CreatedAt,
		c.UpdatedAt,
		c.ID,
		sqlf.Join(changesetSpecColumns, ", "),
	), nil
}

// DeleteChangesetSpec deletes the ChangesetSpec with the given ID.
func (s *Store) DeleteChangesetSpec(ctx context.Context, id int64) error {
	return s.Store.Exec(ctx, sqlf.Sprintf(deleteChangesetSpecQueryFmtstr, id))
}

var deleteChangesetSpecQueryFmtstr = `
-- source: enterprise/internal/campaigns/store_changeset_specs.go:DeleteChangesetSpec
DELETE FROM changeset_specs WHERE id = %s
`

// CountChangesetSpecsOpts captures the query options needed for counting
// ChangesetSpecs.
type CountChangesetSpecsOpts struct {
	CampaignSpecID int64
}

// CountChangesetSpecs returns the number of changeset specs in the database.
func (s *Store) CountChangesetSpecs(ctx context.Context, opts CountChangesetSpecsOpts) (int, error) {
	return s.queryCount(ctx, countChangesetSpecsQuery(&opts))
}

var countChangesetSpecsQueryFmtstr = `
-- source: enterprise/internal/campaigns/store_changeset_specs.go:CountChangesetSpecs
SELECT COUNT(changeset_specs.id)
FROM changeset_specs
INNER JOIN repo ON repo.id = changeset_specs.repo_id
WHERE %s
`

func countChangesetSpecsQuery(opts *CountChangesetSpecsOpts) *sqlf.Query {
	preds := []*sqlf.Query{
		sqlf.Sprintf("repo.deleted_at IS NULL"),
	}

	if opts.CampaignSpecID != 0 {
		cond := sqlf.Sprintf("changeset_specs.campaign_spec_id = %s", opts.CampaignSpecID)
		preds = append(preds, cond)
	}

	if len(preds) == 0 {
		preds = append(preds, sqlf.Sprintf("TRUE"))
	}

	return sqlf.Sprintf(countChangesetSpecsQueryFmtstr, sqlf.Join(preds, "\n AND "))
}

// GetChangesetSpecOpts captures the query options needed for getting a ChangesetSpec
type GetChangesetSpecOpts struct {
	ID     int64
	RandID string
}

// GetChangesetSpec gets a changeset spec matching the given options.
func (s *Store) GetChangesetSpec(ctx context.Context, opts GetChangesetSpecOpts) (*campaigns.ChangesetSpec, error) {
	q := getChangesetSpecQuery(&opts)

	var c campaigns.ChangesetSpec
	err := s.query(ctx, q, func(sc scanner) error {
		return scanChangesetSpec(&c, sc)
	})
	if err != nil {
		return nil, err
	}

	if c.ID == 0 {
		return nil, ErrNoResults
	}

	return &c, nil
}

// GetChangesetSpecByID gets a changeset spec with the given ID.
func (s *Store) GetChangesetSpecByID(ctx context.Context, id int64) (*campaigns.ChangesetSpec, error) {
	return s.GetChangesetSpec(ctx, GetChangesetSpecOpts{ID: id})
}

var getChangesetSpecsQueryFmtstr = `
-- source: enterprise/internal/campaigns/store_changeset_specs.go:GetChangesetSpec
SELECT %s FROM changeset_specs
INNER JOIN repo ON repo.id = changeset_specs.repo_id
WHERE %s
LIMIT 1
`

func getChangesetSpecQuery(opts *GetChangesetSpecOpts) *sqlf.Query {
	preds := []*sqlf.Query{
		sqlf.Sprintf("repo.deleted_at IS NULL"),
	}

	if opts.ID != 0 {
		preds = append(preds, sqlf.Sprintf("changeset_specs.id = %s", opts.ID))
	}

	if opts.RandID != "" {
		preds = append(preds, sqlf.Sprintf("changeset_specs.rand_id = %s", opts.RandID))
	}

	if len(preds) == 0 {
		preds = append(preds, sqlf.Sprintf("TRUE"))
	}

	return sqlf.Sprintf(
		getChangesetSpecsQueryFmtstr,
		sqlf.Join(changesetSpecColumns, ", "),
		sqlf.Join(preds, "\n AND "),
	)
}

// ListChangesetSpecsOpts captures the query options needed for
// listing code mods.
type ListChangesetSpecsOpts struct {
	LimitOpts
	Cursor int64

	CampaignSpecID int64
	RandIDs        []string
}

// ListChangesetSpecs lists ChangesetSpecs with the given filters.
func (s *Store) ListChangesetSpecs(ctx context.Context, opts ListChangesetSpecsOpts) (cs campaigns.ChangesetSpecs, next int64, err error) {
	q := listChangesetSpecsQuery(&opts)

	cs = make(campaigns.ChangesetSpecs, 0, opts.DBLimit())
	err = s.query(ctx, q, func(sc scanner) error {
		var c campaigns.ChangesetSpec
		if err := scanChangesetSpec(&c, sc); err != nil {
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

var listChangesetSpecsQueryFmtstr = `
-- source: enterprise/internal/campaigns/store_changeset_specs.go:ListChangesetSpecs
SELECT %s FROM changeset_specs
INNER JOIN repo ON repo.id = changeset_specs.repo_id
WHERE %s
ORDER BY changeset_specs.id ASC
`

func listChangesetSpecsQuery(opts *ListChangesetSpecsOpts) *sqlf.Query {
	preds := []*sqlf.Query{
		sqlf.Sprintf("changeset_specs.id >= %s", opts.Cursor),
		sqlf.Sprintf("repo.deleted_at IS NULL"),
	}

	if opts.CampaignSpecID != 0 {
		preds = append(preds, sqlf.Sprintf("changeset_specs.campaign_spec_id = %d", opts.CampaignSpecID))
	}

	if len(opts.RandIDs) != 0 {
		ids := make([]*sqlf.Query, 0, len(opts.RandIDs))
		for _, id := range opts.RandIDs {
			if id != "" {
				ids = append(ids, sqlf.Sprintf("%s", id))
			}
		}
		preds = append(preds, sqlf.Sprintf("changeset_specs.rand_id IN (%s)", sqlf.Join(ids, ",")))
	}

	return sqlf.Sprintf(
		listChangesetSpecsQueryFmtstr+opts.LimitOpts.ToDB(),
		sqlf.Join(changesetSpecColumns, ", "),
		sqlf.Join(preds, "\n AND "),
	)
}

// DeleteExpiredChangesetSpecs deletes ChangesetSpecs that have not been
// attached to a CampaignSpec within ChangesetSpecTTL.
func (s *Store) DeleteExpiredChangesetSpecs(ctx context.Context) error {
	expirationTime := s.now().Add(-campaigns.ChangesetSpecTTL)
	q := sqlf.Sprintf(deleteExpiredChangesetSpecsQueryFmtstr, expirationTime)
	return s.Store.Exec(ctx, q)
}

var deleteExpiredChangesetSpecsQueryFmtstr = `
-- source: enterprise/internal/campaigns/store.go:DeleteExpiredChangesetSpecs
DELETE FROM
  changeset_specs cspecs
WHERE
  created_at < %s
AND
(
  -- It was never attached to a campaign_spec
  campaign_spec_id IS NULL

  OR

  (
    -- The campaign_spec is not applied to a campaign
    NOT EXISTS(SELECT 1 FROM campaigns WHERE campaign_spec_id = cspecs.campaign_spec_id)
    AND
    -- and the changeset_spec is not attached to a changeset
    NOT EXISTS(SELECT 1 FROM changesets WHERE current_spec_id = cspecs.id OR previous_spec_id = cspecs.id)
  )
);
`

func scanChangesetSpec(c *campaigns.ChangesetSpec, s scanner) error {
	var spec json.RawMessage

	err := s.Scan(
		&c.ID,
		&c.RandID,
		&c.RawSpec,
		&spec,
		&dbutil.NullInt64{N: &c.CampaignSpecID},
		&c.RepoID,
		&dbutil.NullInt32{N: &c.UserID},
		&c.DiffStatAdded,
		&c.DiffStatChanged,
		&c.DiffStatDeleted,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, "scanning campaign spec")
	}

	c.Spec = new(campaigns.ChangesetSpecDescription)
	if err = json.Unmarshal(spec, c.Spec); err != nil {
		return errors.Wrap(err, "scanChangesetSpec: failed to unmarshal spec")
	}

	return nil
}
