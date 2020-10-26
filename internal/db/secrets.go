package db

import (
	"context"
	"errors"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
)

// secrets provides access to the secrets table.
type secrets struct{}

// ErrorNoSuchSecret is returned when we can't retrieve the specific crypt object that we need.
var ErrorNoSuchSecret = errors.New("failed to find matching secret")

// DeleteByID deletes a secret by a given ID.
func (s *secrets) DeleteByID(ctx context.Context, id int32) error {
	if Mocks.Secrets.DeleteByID != nil {
		return Mocks.Secrets.DeleteByID(ctx, id)
	}

	q := sqlf.Sprintf(
		`DELETE FROM
			secrets
		WHERE
			id=%d
		`, id)

	res, err := dbconn.Global.ExecContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if rows != 1 {
		return ErrorNoSuchSecret
	}
	return err
}

// Delete by key name
func (s *secrets) DeleteByKeyName(ctx context.Context, keyName string) error {
	if Mocks.Secrets.DeleteByKeyName != nil {
		return Mocks.Secrets.DeleteByKeyName(ctx, keyName)
	}

	q := sqlf.Sprintf(
		`DELETE FROM
			secrets
		WHERE
			key_name=%s
		`, keyName)

	_, err := dbconn.Global.ExecContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	return err
}

// Delete the object by sourceType (i.e a repo style object) and the source id.
func (s *secrets) DeleteBySource(ctx context.Context, sourceType string, sourceID int32) error {
	if Mocks.Secrets.DeleteBySource != nil {
		return Mocks.Secrets.DeleteBySource(ctx, sourceType, sourceID)
	}

	q := sqlf.Sprintf(
		`DELETE FROM
			secrets
		WHERE
			source_type=%s AND source_id=%d
		`, sourceType, sourceID)

	_, err := dbconn.Global.ExecContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	return err
}

func (s *secrets) getBySQL(ctx context.Context, query *sqlf.Query) (*types.Secret, error) {
	res, err := dbconn.Global.QueryContext(ctx, query.Query(sqlf.PostgresBindVar), query.Args()...)
	if err != nil {
		return nil, err
	}

	var results []*types.Secret
	defer res.Close()

	for res.Next() {
		var obj types.Secret
		if err := res.Scan(&obj.ID, &obj.SourceType, &obj.SourceID, &obj.KeyName, &obj.Value); err != nil {
			return nil, err
		}
		results = append(results, &obj)
	}

	if len(results) != 1 {
		return nil, ErrorNoSuchSecret
	}

	return results[0], nil
}

// Get by object id
func (s *secrets) GetByID(ctx context.Context, id int32) (*types.Secret, error) {
	if Mocks.Secrets.GetByID != nil {
		return Mocks.Secrets.GetByID(ctx, id)
	}

	q := sqlf.Sprintf(
		`SELECT
			*
		FROM
			secrets
		WHERE
			id=%d
		`, id)

	return s.getBySQL(ctx, q)
}

// Get the secret by the key name - for key/value pair secrets
func (s *secrets) GetByKeyName(ctx context.Context, keyName string) (*types.Secret, error) {
	if Mocks.Secrets.GetByKeyName != nil {
		return Mocks.Secrets.GetByKeyName(ctx, keyName)
	}

	q := sqlf.Sprintf(
		`SELECT
			*
		FROM
			secrets
		WHERE
			key_name=%s
		`, keyName)

	return s.getBySQL(ctx, q)
}

// Get the secret by the sourceType and source id (i.e the specific repo entity)
func (s *secrets) GetBySource(ctx context.Context, sourceType string, sourceID int32) (*types.Secret, error) {
	q := sqlf.Sprintf(
		`SELECT *
		FROM
			secrets
		WHERE
			source_type=%s AND source_id=%d
		`, sourceType, sourceID)

	return s.getBySQL(ctx, q)
}

func (s *secrets) insert(ctx context.Context, query *sqlf.Query) error {
	_, err := dbconn.Global.ExecContext(ctx, query.Query(sqlf.PostgresBindVar), query.Args()...)
	return err
}

// Insert a new key-value secret
func (s *secrets) InsertKeyValue(ctx context.Context, keyName, value string) error {
	q := sqlf.Sprintf(
		`INSERT INTO
			secrets(key_name, value)
		VALUES(%s, %s)
		`, keyName, value)
	return s.insert(ctx, q)
}

// Insert a new secret referenced by another table type
func (s *secrets) InsertSourceTypeValue(ctx context.Context, sourceType string, sourceID int32, value string) error {
	q := sqlf.Sprintf(
		`INSERT INTO
			secrets(source_type, source_id, value)
		VALUES(%s, %d, %s)
		`, sourceType, sourceID, value)
	return s.insert(ctx, q)
}

// Update object id to value
func (s *secrets) UpdateByID(ctx context.Context, id int32, value string) error {
	q := sqlf.Sprintf(
		`UPDATE
			secrets
		SET
			value=%s
		WHERE
			id=%d
		`, value, id)

	res, err := dbconn.Global.ExecContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrorNoSuchSecret
	}

	return nil
}

// Update function for key-value pairs
func (s *secrets) UpdateByKeyName(ctx context.Context, keyName, value string) error {
	q := sqlf.Sprintf(
		`UPDATE
			secrets
		SET
			value=%s
		WHERE
			key_name=%s
		`, value, keyName)

	res, err := dbconn.Global.ExecContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrorNoSuchSecret
	}

	return nil
}

func (s *secrets) UpdateBySource(ctx context.Context, sourceType string, sourceID int32, value string) error {
	q := sqlf.Sprintf(
		`UPDATE
			secrets
		SET
			value=%s
		WHERE
			source_type=%s AND source_id=%d
		`, value, sourceType, sourceID)

	res, err := dbconn.Global.ExecContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrorNoSuchSecret
	}

	return nil
}
