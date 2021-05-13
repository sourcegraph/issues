package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

// DBRelease describes a release of an extension in the extension registry.
type DBRelease struct {
	ID                  int64
	RegistryExtensionID int32
	CreatorUserID       int32
	ReleaseVersion      *string
	ReleaseTag          string
	Manifest            string
	Bundle              *string
	SourceMap           *string
	CreatedAt           time.Time
}

type DBReleases struct {
	*basestore.Store
}

// NewDBReleases returns a new dbReleases backed by the given database.
func NewDBReleases(db dbutil.DB) *DBReleases {
	return &DBReleases{
		Store: basestore.NewWithDB(db, sql.TxOptions{}),
	}
}

// releaseNotFoundError occurs when an extension release is not found in the
// extension registry.
type releaseNotFoundError struct {
	args []interface{}
}

// NotFound implements errcode.NotFounder.
func (err releaseNotFoundError) NotFound() bool { return true }

func (err releaseNotFoundError) Error() string {
	return fmt.Sprintf("registry extension release not found: %v", err.args)
}

var errInvalidJSONInManifest = errors.New("invalid syntax in extension manifest JSON")

// Create creates a new release of an extension in the extension registry. The release.ID and
// release.CreatedAt fields are ignored (they are populated automatically by the database).
func (s *DBReleases) Create(ctx context.Context, release *DBRelease) (id int64, err error) {
	if mocks.releases.Create != nil {
		return mocks.releases.Create(release)
	}

	q := sqlf.Sprintf(
		createQueryFmtstr,
		release.RegistryExtensionID,
		release.CreatorUserID,
		release.ReleaseVersion,
		release.ReleaseTag,
		release.Manifest,
		release.Bundle,
		release.SourceMap,
	)

	if err := s.QueryRow(ctx, q).Scan(&id); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Message == "invalid input syntax for type json" {
				return 0, errInvalidJSONInManifest
			}
		}
		return 0, err
	}
	return id, nil
}

const createQueryFmtstr = `
-- source:enterprise/cmd/frontend/internal/registry/releases_db.go:Create
INSERT INTO registry_extension_releases
	(registry_extension_id, creator_user_id, release_version, release_tag, manifest, bundle, source_map)
VALUES
	(%s, %s, %s, %s, %s, %s, %s)
RETURNING id
`

// GetLatest gets the latest release for the extension with the given release tag (e.g., "release").
// If includeArtifacts is true, it populates the (*dbRelease).{Bundle,SourceMap} fields, which may be large.
func (s *DBReleases) GetLatest(ctx context.Context, registryExtensionID int32, releaseTag string, includeArtifacts bool) (*DBRelease, error) {
	if mocks.releases.GetLatest != nil {
		return mocks.releases.GetLatest(registryExtensionID, releaseTag, includeArtifacts)
	}

	q := sqlf.Sprintf(getLatestQueryFmtstr, includeArtifacts, includeArtifacts, registryExtensionID, releaseTag)

	var r DBRelease
	err := s.QueryRow(ctx, q).Scan(&r.ID, &r.RegistryExtensionID, &r.CreatorUserID, &r.ReleaseVersion, &r.ReleaseTag, &r.Manifest, &r.Bundle, &r.SourceMap, &r.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, releaseNotFoundError{[]interface{}{fmt.Sprintf("latest for registry extension ID %d tag %q", registryExtensionID, releaseTag)}}
		}
		return nil, err
	}
	return &r, nil
}

const getLatestQueryFmtstr = `
-- source:enterprise/cmd/frontend/internal/registry/releases_db.go:GetLatest
SELECT
	id,
	registry_extension_id,
	creator_user_id,
	release_version,
	release_tag,
	manifest,
	CASE WHEN %v::boolean THEN bundle ELSE null END AS bundle,
	CASE WHEN %v::boolean THEN source_map ELSE null END AS source_map,
	created_at
FROM
	registry_extension_releases
WHERE
	registry_extension_id = %d AND
	release_tag = %s
	AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 1
`

// GetLatestBatch gets the latest releases for the extensions with the given release tag
// (e.g., "release"). If includeArtifacts is true, it populates the (*dbRelease).{Bundle,SourceMap}
// fields, which may be large.
func (s *DBReleases) GetLatestBatch(ctx context.Context, registryExtensionIDs []int32, releaseTag string, includeArtifacts bool) ([]*DBRelease, error) {
	if mocks.releases.GetLatestBatch != nil {
		return mocks.releases.GetLatestBatch(registryExtensionIDs, releaseTag, includeArtifacts)
	}

	var ids []*sqlf.Query
	for _, id := range registryExtensionIDs {
		ids = append(ids, sqlf.Sprintf("%s", id))
	}

	if len(ids) == 0 {
		return nil, nil
	}

	q := sqlf.Sprintf(getLatestBatchQueryFmtstr, includeArtifacts, includeArtifacts, sqlf.Join(ids, ","), releaseTag)

	rows, err := s.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var releases []*DBRelease
	for rows.Next() {
		var r DBRelease
		err := rows.Scan(&r.ID, &r.RegistryExtensionID, &r.CreatorUserID, &r.ReleaseVersion, &r.ReleaseTag, &r.Manifest, &r.Bundle, &r.SourceMap, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		releases = append(releases, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return releases, nil
}

const getLatestBatchQueryFmtstr = `
-- source:enterprise/cmd/frontend/internal/registry/releases_db.go:GetLatestBatch
SELECT DISTINCT ON (rer.registry_extension_id)
	rer.id,
	rer.registry_extension_id,
	rer.creator_user_id,
	rer.release_version,
	rer.release_tag,
	rer.manifest,
	CASE WHEN %v::boolean THEN rer.bundle ELSE null END AS bundle,
	CASE WHEN %v::boolean THEN rer.source_map ELSE null END AS source_map,
	rer.created_at
FROM
	registry_extension_releases rer
WHERE
	rer.registry_extension_id IN (%s) AND
	rer.release_tag = %s AND
	rer.deleted_at IS NULL
ORDER BY rer.registry_extension_id, rer.created_at DESC
`

// GetArtifacts gets the bundled JavaScript source file contents and the source map for a release
// (by ID).
func (s *DBReleases) GetArtifacts(ctx context.Context, id int64) (bundle, sourcemap []byte, err error) {
	q := sqlf.Sprintf(getArtifactsQueryFmtstr, id)

	if err := s.QueryRow(ctx, q).Scan(&bundle, &sourcemap); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, releaseNotFoundError{[]interface{}{fmt.Sprintf("registry extension release %d", id)}}
		}
		return nil, nil, err
	}

	if bundle == nil {
		return nil, nil, releaseNotFoundError{[]interface{}{fmt.Sprintf("no bundle for registry extension release %d", id)}}
	}
	return bundle, sourcemap, nil
}

const getArtifactsQueryFmtstr = `
-- source:enterprise/cmd/frontend/internal/registry/releases_db.go:GetArtifacts
SELECT
	bundle, source_map
FROM
	registry_extension_releases
WHERE
	id = %d AND
	deleted_at IS NULL
`

// mockReleases mocks the registry extension releases store.
type mockReleases struct {
	Create         func(release *DBRelease) (int64, error)
	GetLatest      func(registryExtensionID int32, releaseTag string, includeArtifacts bool) (*DBRelease, error)
	GetLatestBatch func(registryExtensionIDs []int32, releaseTag string, includeArtifacts bool) ([]*DBRelease, error)
}
