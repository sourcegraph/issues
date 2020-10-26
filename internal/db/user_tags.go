package db

import (
	"context"

	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
)

// SetTag adds (present=true) or removes (present=false) a tag from the given user's set of tags. An
// error occurs if the user does not exist. Adding a duplicate tag or removing a nonexistent tag is
// not an error.
func (*users) SetTag(ctx context.Context, userID int32, tag string, present bool) error {
	var query string
	if present {
		// Add tag.
		query = `UPDATE users SET tags=CASE WHEN NOT $2::text = ANY(tags) THEN (tags || $2) ELSE tags END WHERE id=$1`
	} else {
		// Remove tag.
		query = `UPDATE users SET tags=array_remove(tags, $2) WHERE id=$1`
	}

	res, err := dbconn.Global.ExecContext(ctx, query, userID, tag)
	if err != nil {
		return err
	}
	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if nrows == 0 {
		return userNotFoundErr{args: []interface{}{userID}}
	}
	return nil
}
