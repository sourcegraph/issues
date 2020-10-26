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

// campaignSpecColumns are used by the campaignSpec related Store methods to insert,
// update and query campaigns.
var campaignSpecColumns = []*sqlf.Query{
	sqlf.Sprintf("campaign_specs.id"),
	sqlf.Sprintf("campaign_specs.rand_id"),
	sqlf.Sprintf("campaign_specs.raw_spec"),
	sqlf.Sprintf("campaign_specs.spec"),
	sqlf.Sprintf("campaign_specs.namespace_user_id"),
	sqlf.Sprintf("campaign_specs.namespace_org_id"),
	sqlf.Sprintf("campaign_specs.user_id"),
	sqlf.Sprintf("campaign_specs.created_at"),
	sqlf.Sprintf("campaign_specs.updated_at"),
}

// campaignSpecInsertColumns is the list of campaign_specs columns that are
// modified when updating/inserting campaign specs.
var campaignSpecInsertColumns = []*sqlf.Query{
	sqlf.Sprintf("rand_id"),
	sqlf.Sprintf("raw_spec"),
	sqlf.Sprintf("spec"),
	sqlf.Sprintf("namespace_user_id"),
	sqlf.Sprintf("namespace_org_id"),
	sqlf.Sprintf("user_id"),
	sqlf.Sprintf("created_at"),
	sqlf.Sprintf("updated_at"),
}

const campaignSpecInsertColsFmt = `(%s, %s, %s, %s, %s, %s, %s, %s)`

// CreateCampaignSpec creates the given CampaignSpec.
func (s *Store) CreateCampaignSpec(ctx context.Context, c *campaigns.CampaignSpec) error {
	q, err := s.createCampaignSpecQuery(c)
	if err != nil {
		return err
	}
	return s.query(ctx, q, func(sc scanner) error { return scanCampaignSpec(c, sc) })
}

var createCampaignSpecQueryFmtstr = `
-- source: enterprise/internal/campaigns/store_campaign_specs.go:CreateCampaignSpec
INSERT INTO campaign_specs (%s)
VALUES ` + campaignSpecInsertColsFmt + `
RETURNING %s`

func (s *Store) createCampaignSpecQuery(c *campaigns.CampaignSpec) (*sqlf.Query, error) {
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
		createCampaignSpecQueryFmtstr,
		sqlf.Join(campaignSpecInsertColumns, ", "),
		c.RandID,
		c.RawSpec,
		spec,
		nullInt32Column(c.NamespaceUserID),
		nullInt32Column(c.NamespaceOrgID),
		nullInt32Column(c.UserID),
		c.CreatedAt,
		c.UpdatedAt,
		sqlf.Join(campaignSpecColumns, ", "),
	), nil
}

// UpdateCampaignSpec updates the given CampaignSpec.
func (s *Store) UpdateCampaignSpec(ctx context.Context, c *campaigns.CampaignSpec) error {
	q, err := s.updateCampaignSpecQuery(c)
	if err != nil {
		return err
	}

	return s.query(ctx, q, func(sc scanner) error {
		return scanCampaignSpec(c, sc)
	})
}

var updateCampaignSpecQueryFmtstr = `
-- source: enterprise/internal/campaigns/store_campaign_specs.go:UpdateCampaignSpec
UPDATE campaign_specs
SET (%s) = ` + campaignSpecInsertColsFmt + `
WHERE id = %s
RETURNING %s`

func (s *Store) updateCampaignSpecQuery(c *campaigns.CampaignSpec) (*sqlf.Query, error) {
	spec, err := jsonbColumn(c.Spec)
	if err != nil {
		return nil, err
	}

	c.UpdatedAt = s.now()

	return sqlf.Sprintf(
		updateCampaignSpecQueryFmtstr,
		sqlf.Join(campaignSpecInsertColumns, ", "),
		c.RandID,
		c.RawSpec,
		spec,
		nullInt32Column(c.NamespaceUserID),
		nullInt32Column(c.NamespaceOrgID),
		nullInt32Column(c.UserID),
		c.CreatedAt,
		c.UpdatedAt,
		c.ID,
		sqlf.Join(campaignSpecColumns, ", "),
	), nil
}

// DeleteCampaignSpec deletes the CampaignSpec with the given ID.
func (s *Store) DeleteCampaignSpec(ctx context.Context, id int64) error {
	return s.Store.Exec(ctx, sqlf.Sprintf(deleteCampaignSpecQueryFmtstr, id))
}

var deleteCampaignSpecQueryFmtstr = `
-- source: enterprise/internal/campaigns/store_campaign_specs.go:DeleteCampaignSpec
DELETE FROM campaign_specs WHERE id = %s
`

// CountCampaignSpecs returns the number of code mods in the database.
func (s *Store) CountCampaignSpecs(ctx context.Context) (int, error) {
	return s.queryCount(ctx, sqlf.Sprintf(countCampaignSpecsQueryFmtstr))
}

var countCampaignSpecsQueryFmtstr = `
-- source: enterprise/internal/campaigns/store_campaign_specs.go:CountCampaignSpecs
SELECT COUNT(id)
FROM campaign_specs
`

// GetCampaignSpecOpts captures the query options needed for getting a CampaignSpec
type GetCampaignSpecOpts struct {
	ID     int64
	RandID string
}

// GetCampaignSpec gets a code mod matching the given options.
func (s *Store) GetCampaignSpec(ctx context.Context, opts GetCampaignSpecOpts) (*campaigns.CampaignSpec, error) {
	q := getCampaignSpecQuery(&opts)

	var c campaigns.CampaignSpec
	err := s.query(ctx, q, func(sc scanner) (err error) {
		return scanCampaignSpec(&c, sc)
	})
	if err != nil {
		return nil, err
	}

	if c.ID == 0 {
		return nil, ErrNoResults
	}

	return &c, nil
}

var getCampaignSpecsQueryFmtstr = `
-- source: enterprise/internal/campaigns/store_campaign_specs.go:GetCampaignSpec
SELECT %s FROM campaign_specs
WHERE %s
LIMIT 1
`

func getCampaignSpecQuery(opts *GetCampaignSpecOpts) *sqlf.Query {
	var preds []*sqlf.Query
	if opts.ID != 0 {
		preds = append(preds, sqlf.Sprintf("id = %s", opts.ID))
	}

	if opts.RandID != "" {
		preds = append(preds, sqlf.Sprintf("rand_id = %s", opts.RandID))
	}

	if len(preds) == 0 {
		preds = append(preds, sqlf.Sprintf("TRUE"))
	}

	return sqlf.Sprintf(
		getCampaignSpecsQueryFmtstr,
		sqlf.Join(campaignSpecColumns, ", "),
		sqlf.Join(preds, "\n AND "),
	)
}

// ListCampaignSpecsOpts captures the query options needed for
// listing code mods.
type ListCampaignSpecsOpts struct {
	LimitOpts
	Cursor int64
}

// ListCampaignSpecs lists CampaignSpecs with the given filters.
func (s *Store) ListCampaignSpecs(ctx context.Context, opts ListCampaignSpecsOpts) (cs []*campaigns.CampaignSpec, next int64, err error) {
	q := listCampaignSpecsQuery(&opts)

	cs = make([]*campaigns.CampaignSpec, 0, opts.DBLimit())
	err = s.query(ctx, q, func(sc scanner) error {
		var c campaigns.CampaignSpec
		if err := scanCampaignSpec(&c, sc); err != nil {
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

var listCampaignSpecsQueryFmtstr = `
-- source: enterprise/internal/campaigns/store_campaign_specs.go:ListCampaignSpecs
SELECT %s FROM campaign_specs
WHERE %s
ORDER BY id ASC
`

func listCampaignSpecsQuery(opts *ListCampaignSpecsOpts) *sqlf.Query {
	preds := []*sqlf.Query{
		sqlf.Sprintf("id >= %s", opts.Cursor),
	}

	return sqlf.Sprintf(
		listCampaignSpecsQueryFmtstr+opts.LimitOpts.ToDB(),
		sqlf.Join(campaignSpecColumns, ", "),
		sqlf.Join(preds, "\n AND "),
	)
}

// DeleteExpiredCampaignSpecs deletes CampaignSpecs that have not been attached
// to a Campaign within CampaignSpecTTL.
func (s *Store) DeleteExpiredCampaignSpecs(ctx context.Context) error {
	expirationTime := s.now().Add(-campaigns.CampaignSpecTTL)
	q := sqlf.Sprintf(deleteExpiredCampaignSpecsQueryFmtstr, expirationTime)

	return s.Store.Exec(ctx, q)
}

var deleteExpiredCampaignSpecsQueryFmtstr = `
-- source: enterprise/internal/campaigns/store.go:DeleteExpiredCampaignSpecs
DELETE FROM
  campaign_specs
WHERE
  created_at < %s
AND
NOT EXISTS (
  SELECT 1 FROM campaigns WHERE campaign_spec_id = campaign_specs.id
)
AND NOT EXISTS (
  SELECT 1 FROM changeset_specs WHERE campaign_spec_id = campaign_specs.id
);
`

func scanCampaignSpec(c *campaigns.CampaignSpec, s scanner) error {
	var spec json.RawMessage

	err := s.Scan(
		&c.ID,
		&c.RandID,
		&c.RawSpec,
		&spec,
		&dbutil.NullInt32{N: &c.NamespaceUserID},
		&dbutil.NullInt32{N: &c.NamespaceOrgID},
		&dbutil.NullInt32{N: &c.UserID},
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, "scanning campaign spec")
	}

	if err = json.Unmarshal(spec, &c.Spec); err != nil {
		return errors.Wrap(err, "scanCampaignSpec: failed to unmarshal spec")
	}

	return nil
}
