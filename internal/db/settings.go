package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/jsonx"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

type settings struct{}

func (o *settings) CreateIfUpToDate(ctx context.Context, subject api.SettingsSubject, lastID *int32, authorUserID *int32, contents string) (latestSetting *api.Settings, err error) {
	if Mocks.Settings.CreateIfUpToDate != nil {
		return Mocks.Settings.CreateIfUpToDate(ctx, subject, lastID, authorUserID, contents)
	}

	if strings.TrimSpace(contents) == "" {
		return nil, fmt.Errorf("blank settings are invalid (you can clear the settings by entering an empty JSON object: {})")
	}

	// Validate JSON syntax before saving.
	if _, errs := jsonx.Parse(contents, jsonx.ParseOptions{Comments: true, TrailingCommas: true}); len(errs) > 0 {
		return nil, fmt.Errorf("invalid settings JSON: %v", errs)
	}

	s := api.Settings{
		Subject:      subject,
		AuthorUserID: authorUserID,
		Contents:     contents,
	}

	tx, err := dbconn.Global.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			rollErr := tx.Rollback()
			if rollErr != nil {
				err = multierror.Append(err, rollErr)
			}
			return
		}
		err = tx.Commit()
	}()

	latestSetting, err = o.getLatest(ctx, tx, subject)
	if err != nil {
		return nil, err
	}

	creatorIsUpToDate := latestSetting != nil && lastID != nil && latestSetting.ID == *lastID
	if latestSetting == nil || creatorIsUpToDate {
		err := tx.QueryRow(
			"INSERT INTO settings(org_id, user_id, author_user_id, contents) VALUES($1, $2, $3, $4) RETURNING id, created_at",
			s.Subject.Org, s.Subject.User, s.AuthorUserID, s.Contents).Scan(&s.ID, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		latestSetting = &s
	}

	return latestSetting, nil
}

func (o *settings) GetLatest(ctx context.Context, subject api.SettingsSubject) (*api.Settings, error) {
	if Mocks.Settings.GetLatest != nil {
		return Mocks.Settings.GetLatest(ctx, subject)
	}

	return o.getLatest(ctx, dbconn.Global, subject)
}

// ListAll lists ALL settings (across all users, orgs, etc).
//
// If impreciseSubstring is given, only settings whose raw JSONC string contains the substring are
// returned. This is only intended for use by migration code that needs to update settings, where
// limiting to matching settings can significantly narrow the amount of work necessary. For example,
// a migration to rename a settings property `foo` to `bar` would narrow the search space by only
// processing settings that contain the string `foo` (which will yield false positives, but that's
// acceptable because the migration shouldn't modify those results anyway).
//
// 🚨 SECURITY: This method does NOT verify the user is an admin. The caller is
// responsible for ensuring this or that the response never makes it to a user.
func (o *settings) ListAll(ctx context.Context, impreciseSubstring string) (_ []*api.Settings, err error) {
	tr, ctx := trace.New(ctx, "db.Settings.ListAll", "")
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	q := sqlf.Sprintf(`
		WITH q AS (
			SELECT DISTINCT
				ON (org_id, user_id, author_user_id)
				id, org_id, user_id, author_user_id, contents, created_at
				FROM settings
				ORDER BY org_id, user_id, author_user_id, id DESC
		)
		SELECT q.id, q.org_id, q.user_id, CASE WHEN users.deleted_at IS NULL THEN q.author_user_id ELSE NULL END, q.contents, q.created_at
		FROM q
		LEFT JOIN users ON users.id=q.author_user_id
		WHERE contents LIKE %s
		ORDER BY q.org_id, q.user_id, q.author_user_id, q.id DESC
	`, "%"+impreciseSubstring+"%")
	rows, err := dbconn.Global.QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return nil, err
	}
	return o.parseQueryRows(ctx, rows)
}

func (o *settings) getLatest(ctx context.Context, queryTarget queryable, subject api.SettingsSubject) (*api.Settings, error) {
	var cond *sqlf.Query
	switch {
	case subject.Org != nil:
		cond = sqlf.Sprintf("org_id=%d", *subject.Org)
	case subject.User != nil:
		cond = sqlf.Sprintf("user_id=%d AND EXISTS (SELECT NULL FROM users WHERE id=%d AND deleted_at IS NULL)", *subject.User, *subject.User)
	default:
		// No org and no user represents global site settings.
		cond = sqlf.Sprintf("user_id IS NULL AND org_id IS NULL")
	}

	q := sqlf.Sprintf(`
		SELECT s.id, s.org_id, s.user_id, CASE WHEN users.deleted_at IS NULL THEN s.author_user_id ELSE NULL END, s.contents, s.created_at FROM settings s
		LEFT JOIN users ON users.id=s.author_user_id
		WHERE %s
		ORDER BY id DESC LIMIT 1`, cond)
	rows, err := queryTarget.QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return nil, err
	}
	settings, err := o.parseQueryRows(ctx, rows)
	if err != nil {
		return nil, err
	}
	if len(settings) != 1 {
		// No configuration has been set for this subject yet.
		return nil, nil
	}
	if settings[0].Contents == "" {
		// On some instances user, org, and global settings are an invalid
		// empty string / null.
		//
		// This happens particularly on instances that ran old versions of
		// Sourcegraph where we didn't enforce that settings contents had to be
		// non-empty for correctness.
		settings[0].Contents = "{}"
	}
	return settings[0], nil
}

// queryable allows us to reuse the same logic for certain operations both
// inside and outside an explicit transaction.
type queryable interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

func (o *settings) parseQueryRows(ctx context.Context, rows *sql.Rows) ([]*api.Settings, error) {
	settings := []*api.Settings{}
	defer rows.Close()
	for rows.Next() {
		s := api.Settings{}
		err := rows.Scan(&s.ID, &s.Subject.Org, &s.Subject.User, &s.AuthorUserID, &s.Contents, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		if s.Subject.Org == nil && s.Subject.User == nil {
			s.Subject.Site = true
		}
		settings = append(settings, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return settings, nil
}
