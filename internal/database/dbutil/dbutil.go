package dbutil

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgconn"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
)

// Transaction calls f within a transaction, rolling back if any error is
// returned by the function.
func Transaction(ctx context.Context, db *sql.DB, f func(tx *sql.Tx) error) (err error) {
	finish := func(tx *sql.Tx) {
		if err != nil {
			if err2 := tx.Rollback(); err2 != nil {
				err = multierror.Append(err, err2)
			}
			return
		}
		err = tx.Commit()
	}

	span, ctx := ot.StartSpanFromContext(ctx, "Transaction")
	defer func() {
		if err != nil {
			ext.Error.Set(span, true)
			span.SetTag("err", err.Error())
		}
		span.Finish()
	}()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer finish(tx)
	return f(tx)
}

// A DB captures the essential method of a sql.DB: QueryContext, ExecContext and QueryRowContext.
type DB interface {
	QueryContext(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// A Tx captures the essential methods of a sql.Tx.
type Tx interface {
	Rollback() error
	Commit() error
}

// A TxBeginner captures BeginTx method of a sql.DB
type TxBeginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func IsPostgresError(err error, codename string) bool {
	e, ok := errors.Cause(err).(*pgconn.PgError)
	return ok && e.Code == codename
}

// NullTime represents a time.Time that may be null. nullTime implements the
// sql.Scanner interface so it can be used as a scan destination, similar to
// sql.NullString. When the scanned value is null, Time is set to the zero value.
type NullTime struct{ *time.Time }

// Scan implements the Scanner interface.
func (nt *NullTime) Scan(value interface{}) error {
	*nt.Time, _ = value.(time.Time)
	return nil
}

// Value implements the driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if nt.Time == nil {
		return nil, nil
	}
	return *nt.Time, nil
}

// NullString represents a string that may be null. NullString implements the
// sql.Scanner interface so it can be used as a scan destination, similar to
// sql.NullString. When the scanned value is null, String is set to the zero value.
type NullString struct{ S *string }

// NewNullString returns a NullString treating zero value as null.
func NewNullString(s string) NullString {
	if s == "" {
		return NullString{}
	}
	return NullString{S: &s}
}

// Scan implements the Scanner interface.
func (nt *NullString) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		*nt.S = string(v)
	case string:
		*nt.S = v
	}
	return nil
}

// Value implements the driver Valuer interface.
func (nt NullString) Value() (driver.Value, error) {
	if nt.S == nil {
		return nil, nil
	}
	return *nt.S, nil
}

// NullInt32 represents an int32 that may be null. NullInt32 implements the
// sql.Scanner interface so it can be used as a scan destination, similar to
// sql.NullString. When the scanned value is null, int32 is set to the zero value.
type NullInt32 struct{ N *int32 }

// Scan implements the Scanner interface.
func (n *NullInt32) Scan(value interface{}) error {
	switch value := value.(type) {
	case int64:
		*n.N = int32(value)
	case int32:
		*n.N = value
	case nil:
		return nil
	default:
		return fmt.Errorf("value is not int64: %T", value)
	}
	return nil
}

// Value implements the driver Valuer interface.
func (n NullInt32) Value() (driver.Value, error) {
	if n.N == nil {
		return nil, nil
	}
	return *n.N, nil
}

// NullInt64 represents an int64 that may be null. NullInt64 implements the
// sql.Scanner interface so it can be used as a scan destination, similar to
// sql.NullString. When the scanned value is null, int64 is set to the zero value.
type NullInt64 struct{ N *int64 }

// NewNullInt64 returns a NullInt64 treating zero value as null.
func NewNullInt64(i int64) NullInt64 {
	if i == 0 {
		return NullInt64{}
	}
	return NullInt64{N: &i}
}

// Scan implements the Scanner interface.
func (n *NullInt64) Scan(value interface{}) error {
	switch value := value.(type) {
	case int64:
		*n.N = value
	case int32:
		*n.N = int64(value)
	case nil:
		return nil
	default:
		return fmt.Errorf("value is not int64: %T", value)
	}
	return nil
}

// Value implements the driver Valuer interface.
func (n NullInt64) Value() (driver.Value, error) {
	if n.N == nil {
		return nil, nil
	}
	return *n.N, nil
}

// NullInt represents an int that may be null. NullInt implements the
// sql.Scanner interface so it can be used as a scan destination, similar to
// sql.NullString. When the scanned value is null, int is set to the zero value.
type NullInt struct{ N *int }

// NewNullInt returns a NullInt treating zero value as null.
func NewNullInt(i int) NullInt {
	if i == 0 {
		return NullInt{}
	}
	return NullInt{N: &i}
}

// Scan implements the Scanner interface.
func (n *NullInt) Scan(value interface{}) error {
	switch value := value.(type) {
	case int64:
		*n.N = int(value)
	case int32:
		*n.N = int(value)
	case nil:
		return nil
	default:
		return fmt.Errorf("value is not int: %T", value)
	}
	return nil
}

// Value implements the driver Valuer interface.
func (n NullInt) Value() (driver.Value, error) {
	if n.N == nil {
		return nil, nil
	}
	return *n.N, nil
}

// JSONInt64Set represents an int64 set as a JSONB object where the keys are
// the ids and the values are null. It implements the sql.Scanner interface so
// it can be used as a scan destination, similar to
// sql.NullString.
type JSONInt64Set struct{ Set *[]int64 }

// Scan implements the Scanner interface.
func (n *JSONInt64Set) Scan(value interface{}) error {
	set := make(map[int64]*struct{})

	switch value := value.(type) {
	case nil:
	case []byte:
		if err := json.Unmarshal(value, &set); err != nil {
			return err
		}
	default:
		return fmt.Errorf("value is not []byte: %T", value)
	}

	if *n.Set == nil {
		*n.Set = make([]int64, 0, len(set))
	} else {
		*n.Set = (*n.Set)[:0]
	}

	for id := range set {
		*n.Set = append(*n.Set, id)
	}

	return nil
}

// Value implements the driver Valuer interface.
func (n JSONInt64Set) Value() (driver.Value, error) {
	if n.Set == nil {
		return nil, nil
	}
	return *n.Set, nil
}

// NullJSONRawMessage represents a json.RawMessage that may be null. NullJSONRawMessage implements the
// sql.Scanner interface so it can be used as a scan destination, similar to
// sql.NullString. When the scanned value is null, Raw is left as nil.
type NullJSONRawMessage struct {
	Raw json.RawMessage
}

// Scan implements the Scanner interface.
func (n *NullJSONRawMessage) Scan(value interface{}) error {
	switch value := value.(type) {
	case nil:
	case []byte:
		// We make a copy here because the given value could be reused by
		// the SQL driver
		n.Raw = make([]byte, len(value))
		copy(n.Raw, value)
	default:
		return fmt.Errorf("value is not []byte: %T", value)
	}

	return nil
}

// Value implements the driver Valuer interface.
func (n *NullJSONRawMessage) Value() (driver.Value, error) {
	return n.Raw, nil
}

// CommitBytea represents a hex-encoded string that is efficiently encoded in Postgres and should
// be used in place of a text or varchar type when the table is large (e.g. a record per commit).
type CommitBytea string

// Scan implements the Scanner interface.
func (c *CommitBytea) Scan(value interface{}) error {
	switch value := value.(type) {
	case nil:
	case []byte:
		*c = CommitBytea(hex.EncodeToString(value))
	default:
		return fmt.Errorf("value is not []byte: %T", value)
	}

	return nil
}

// Value implements the driver Valuer interface.
func (c CommitBytea) Value() (driver.Value, error) {
	return hex.DecodeString(string(c))
}

func PostgresDSN(prefix, currentUser string, getenv func(string) string) string {
	if prefix != "" {
		prefix = fmt.Sprintf("%s_", strings.ToUpper(prefix))
	}

	env := func(name string) string {
		return getenv(prefix + name)
	}

	// PGDATASOURCE is a sourcegraph specific variable for just setting the DSN
	if dsn := env("PGDATASOURCE"); dsn != "" {
		return dsn
	}

	// TODO match logic in lib/pq
	// https://sourcegraph.com/github.com/lib/pq@d6156e141ac6c06345c7c73f450987a9ed4b751f/-/blob/connector.go#L42
	dsn := &url.URL{
		Scheme: "postgres",
		Host:   "127.0.0.1:5432",
	}

	// Username preference: PGUSER, $USER, postgres
	username := "postgres"
	if currentUser != "" {
		username = currentUser
	}
	if user := env("PGUSER"); user != "" {
		username = user
	}

	if password := env("PGPASSWORD"); password != "" {
		dsn.User = url.UserPassword(username, password)
	} else {
		dsn.User = url.User(username)
	}

	if host := env("PGHOST"); host != "" {
		dsn.Host = host
	}

	if port := env("PGPORT"); port != "" {
		dsn.Host += ":" + port
	}

	if db := env("PGDATABASE"); db != "" {
		dsn.Path = db
	}

	if sslmode := env("PGSSLMODE"); sslmode != "" {
		qry := dsn.Query()
		qry.Set("sslmode", sslmode)
		dsn.RawQuery = qry.Encode()
	}

	return dsn.String()
}

// Scanner captures the Scan method of sql.Rows and sql.Row
type Scanner interface {
	Scan(dst ...interface{}) error
}

// A ScanFunc scans one or more rows from a scanner, returning
// the last id column scanned and the count of scanned rows.
type ScanFunc func(Scanner) (last, count int64, err error)
