package codemonitors

import (
	"context"
	"database/sql"

	"github.com/cockroachdb/errors"
	"github.com/graph-gophers/graphql-go"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
)

type Recipient struct {
	ID              int64
	Email           int64
	NamespaceUserID *int32
	NamespaceOrgID  *int32
}

func (s *Store) CreateRecipients(ctx context.Context, recipients []graphql.ID, emailID int64) (err error) {
	for _, r := range recipients {
		err = s.createRecipient(ctx, r, emailID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteRecipients(ctx context.Context, emailID int64) (err error) {
	var q *sqlf.Query
	q, err = deleteRecipientsQuery(ctx, emailID)
	if err != nil {
		return err
	}
	err = s.Exec(ctx, q)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) RecipientsForEmailIDInt64(ctx context.Context, emailID int64, args *graphqlbackend.ListRecipientsArgs) ([]*Recipient, error) {
	q, err := readRecipientQuery(ctx, emailID, args)
	if err != nil {
		return nil, err
	}
	rows, err := s.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ms, err := scanRecipients(rows)
	if err != nil {
		return nil, err
	}
	return ms, nil
}

func scanRecipients(rows *sql.Rows) (ms []*Recipient, err error) {
	for rows.Next() {
		m := &Recipient{}
		if err := rows.Scan(
			&m.ID,
			&m.Email,
			&m.NamespaceUserID,
			&m.NamespaceOrgID,
		); err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	err = rows.Close()
	if err != nil {
		return nil, err
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return ms, nil
}

const allRecipientsForEmailIDInt64FmtStr = `
SELECT id, email, namespace_user_id, namespace_org_id
FROM cm_recipients
WHERE email = %s
`

func (s *Store) AllRecipientsForEmailIDInt64(ctx context.Context, emailID int64) (rs []*Recipient, err error) {
	var rows *sql.Rows
	rows, err = s.Query(ctx, sqlf.Sprintf(allRecipientsForEmailIDInt64FmtStr, emailID))
	if err != nil {
		return nil, errors.Errorf("store.AllRecipientsForEmailIDInt64: %w", err)
	}
	defer func() { err = rows.Close() }()
	return scanRecipients(rows)
}

const createRecipientFmtStr = `
INSERT INTO cm_recipients (email, namespace_user_id, namespace_org_id)
VALUES (%s,%s,%s)`

func (s *Store) createRecipient(ctx context.Context, recipient graphql.ID, emailID int64) (err error) {
	var (
		userID int32
		orgID  int32
	)
	err = graphqlbackend.UnmarshalNamespaceID(recipient, &userID, &orgID)
	if err != nil {
		return err
	}
	return s.Exec(ctx, sqlf.Sprintf(createRecipientFmtStr, emailID, nilOrInt32(userID), nilOrInt32(orgID)))
}

const totalCountRecipientsFmtStr = `
SELECT COUNT(*)
FROM cm_recipients
WHERE email = %s
`

func (s *Store) TotalCountRecipients(ctx context.Context, emailID int64) (count int32, err error) {
	err = s.QueryRow(ctx, sqlf.Sprintf(totalCountRecipientsFmtStr, emailID)).Scan(&count)
	return count, err
}

const deleteRecipientFmtStr = `DELETE FROM cm_recipients WHERE email = %s`

func deleteRecipientsQuery(ctx context.Context, emailId int64) (*sqlf.Query, error) {
	return sqlf.Sprintf(
		deleteRecipientFmtStr,
		emailId,
	), nil
}

const readRecipientQueryFmtStr = `
SELECT id, email, namespace_user_id, namespace_org_id
FROM cm_recipients
WHERE email = %s
AND id > %s
ORDER BY id ASC
LIMIT %s;
`

func readRecipientQuery(ctx context.Context, emailId int64, args *graphqlbackend.ListRecipientsArgs) (*sqlf.Query, error) {
	after, err := unmarshalAfter(args.After)
	if err != nil {
		return nil, err
	}
	return sqlf.Sprintf(
		readRecipientQueryFmtStr,
		emailId,
		after,
		args.First,
	), nil
}

func nilOrInt32(n int32) *int32 {
	if n == 0 {
		return nil
	}
	return &n
}
