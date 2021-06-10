// Package dbtesting provides database test helpers.
package dbtesting

import (
	"context"
	"database/sql"
	"hash/fnv"
	"io"
	"strconv"
	"testing"
)

// MockHashPassword if non-nil is used instead of database.hashPassword. This is useful
// when running tests since we can use a faster implementation.
var MockHashPassword func(password string) (sql.NullString, error)
var MockValidPassword func(hash, password string) bool

func useFastPasswordMocks() {
	// We can't care about security in tests, we care about speed.
	MockHashPassword = func(password string) (sql.NullString, error) {
		h := fnv.New64()
		_, _ = io.WriteString(h, password)
		return sql.NullString{Valid: true, String: strconv.FormatUint(h.Sum64(), 16)}, nil
	}
	MockValidPassword = func(hash, password string) bool {
		h := fnv.New64()
		_, _ = io.WriteString(h, password)
		return hash == strconv.FormatUint(h.Sum64(), 16)
	}
}

func setupGlobalTestDB(t testing.TB) {
	useFastPasswordMocks()

	if testing.Short() {
		t.Skip()
	}
}

// MockDB implements the dbutil.DB interface and is intended to be used
// in tests that require the database handle but never call it.
type MockDB struct{}

func (db *MockDB) QueryContext(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
	panic("mock db methods are not supposed to be called")
}

func (db *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	panic("mock db methods are not supposed to be called")
}

func (db *MockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	panic("mock db methods are not supposed to be called")
}
