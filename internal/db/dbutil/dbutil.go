package dbutil

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	// Register driver
	"github.com/lib/pq"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
	codeintelMigrations "github.com/sourcegraph/sourcegraph/migrations/codeintel"
	frontendMigrations "github.com/sourcegraph/sourcegraph/migrations/frontend"
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

// A DB captures the essential method of a sql.DB: QueryContext.
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

// NewDB returns a new *sql.DB from the given dsn (data source name).
func NewDB(dsn, app string) (*sql.DB, error) {
	cfg, err := url.Parse(dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse dsn")
	}

	qry := cfg.Query()

	// Force PostgreSQL session timezone to UTC.
	qry.Set("timezone", "UTC")

	// Force application name.
	qry.Set("application_name", app)

	// Set max open and idle connections
	maxOpen, _ := strconv.Atoi(qry.Get("max_conns"))
	if maxOpen == 0 {
		maxOpen = 30
	}
	qry.Del("max_conns")

	cfg.RawQuery = qry.Encode()
	db, err := sql.Open("postgres", cfg.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	if err := db.Ping(); err != nil {
		return nil, errors.Wrap(err, "failed to ping database")
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxOpen)
	db.SetConnMaxLifetime(time.Minute)

	return db, nil
}

// databases configures the migrations we want based on a database name. This
// configuration includes the name of the migration version table as well as
// the raw migration assets to run to migrate the target schema to a new version.
var databases = map[string]struct {
	MigrationsTable string
	Resource        *bindata.AssetSource
}{
	"frontend": {
		MigrationsTable: "schema_migrations",
		Resource:        bindata.Resource(frontendMigrations.AssetNames(), frontendMigrations.Asset),
	},
	"codeintel": {
		MigrationsTable: "codeintel_schema_migrations",
		Resource:        bindata.Resource(codeintelMigrations.AssetNames(), codeintelMigrations.Asset),
	},
}

// DatabaseNames returns the list of database names (configured via `dbutil.databases`)..
var DatabaseNames = func() []string {
	var names []string
	for databaseName := range databases {
		names = append(names, databaseName)
	}

	return names
}()

// MigrationTables returns the list of migration table names (configured via `dbutil.databases`).
var MigrationTables = func() []string {
	var migrationTables []string
	for _, db := range databases {
		migrationTables = append(migrationTables, db.MigrationsTable)
	}

	return migrationTables
}()

// NewMigrate returns a new configured migration object for the given database name. This database
// name must be present in the `dbutil.databases` map. This migration can be subsequently run by
// invoking `dbutil.DoMigrate`.
func NewMigrate(db *sql.DB, databaseName string) (*migrate.Migrate, error) {
	schemaData, ok := databases[databaseName]
	if !ok {
		return nil, fmt.Errorf("unknown database '%s'", databaseName)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: schemaData.MigrationsTable,
	})
	if err != nil {
		return nil, err
	}

	d, err := bindata.WithInstance(schemaData.Resource)
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithInstance("go-bindata", d, "postgres", driver)
	if err != nil {
		return nil, err
	}

	// In case another process was faster and runs migrations, we will wait
	// this long
	m.LockTimeout = 5 * time.Minute
	if os.Getenv("LOG_MIGRATE_TO_STDOUT") != "" {
		m.Log = stdoutLogger{}
	}

	return m, nil
}

// DoMigrate runs all up migrations.
func DoMigrate(m *migrate.Migrate) (err error) {
	err = m.Up()
	if err == nil || err == migrate.ErrNoChange {
		return nil
	}

	if os.IsNotExist(err) {
		// This should only happen if the DB is ahead of the migrations available
		version, dirty, verr := m.Version()
		if verr != nil {
			return verr
		}
		if dirty { // this shouldn't happen, but checking anyways
			return err
		}
		log15.Warn("WARNING: Detected an old version of Sourcegraph. The database has migrated to a newer version. If you have applied a rollback, this is expected and you can ignore this warning. If not, please contact support@sourcegraph.com for further assistance.", "db_version", version)
		return nil
	}
	return err
}

type stdoutLogger struct{}

func (stdoutLogger) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}
func (logger stdoutLogger) Verbose() bool {
	return true
}

func IsPostgresError(err error, codename string) bool {
	e, ok := errors.Cause(err).(*pq.Error)
	return ok && e.Code.Name() == codename
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
