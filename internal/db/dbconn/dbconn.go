// Package dbconn provides functionality to connect to our DB and migrate it.
//
// Most services should connect to the frontend for DB access instead, using
// api.InternalClient.
package dbconn

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gchaincl/sqlhooks"
	"github.com/inconshreveable/log15"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

var (
	// Global is the global DB connection.
	// Only use this after a call to SetupGlobalConnection.
	Global *sql.DB

	defaultDataSource      = env.Get("PGDATASOURCE", "", "Default dataSource to pass to Postgres. See https://pkg.go.dev/github.com/jackc/pgx for more information.")
	defaultApplicationName = env.Get("PGAPPLICATIONNAME", "sourcegraph", "The value of application_name appended to dataSource")
	// Ensure all time instances have their timezones set to UTC.
	// https://github.com/golang/go/blob/7eb31d999cf2769deb0e7bdcafc30e18f52ceb48/src/time/zoneinfo_unix.go#L29-L34
	_ = env.Ensure("TZ", "UTC", "timezone used by time instances")
)

// SetupGlobalConnection connects to the given data source and stores the handle
// globally.
//
// Note: github.com/jackc/pgx parses the environment as well. This function will
// also use the value of PGDATASOURCE if supplied and dataSource is the empty
// string.
// Deprecated: Use New instead.
func SetupGlobalConnection(dataSource string) (err error) {
	Global, err = New(dataSource, "_app")
	return err
}

// New connects to the given data source and returns the handle.
//
// Note: github.com/jackc/pgx parses the environment as well. This function will
// also use the value of PGDATASOURCE if supplied and dataSource is the empty
// string.
func New(dataSource, dbNameSuffix string) (*sql.DB, error) {
	db, err := NewRaw(dataSource)
	if err != nil {
		return nil, err
	}

	registerPrometheusCollector(db, dbNameSuffix)
	configureConnectionPool(db)
	return db, nil
}

// NewRaw connects to the given data source and returns the handle.
//
// Prefer to call New as it also configures a connection pool and metrics.
// Use this method only in internal utilities (such as schemadoc).
func NewRaw(dataSource string) (*sql.DB, error) {
	cfg, err := buildConfig(dataSource)
	if err != nil {
		return nil, err
	}

	db, err := openDBWithStartupWait(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "DB not available")
	}
	return db, nil
}

var startupTimeout = func() time.Duration {
	str := env.Get("DB_STARTUP_TIMEOUT", "10s", "keep trying for this long to connect to PostgreSQL database before failing")
	d, err := time.ParseDuration(str)
	if err != nil {
		log.Fatalln("DB_STARTUP_TIMEOUT:", err)
	}
	return d
}()

// buildConfig takes either a Postgres connection string or connection URI,
// parses it, and returns a config with additional parameters.
func buildConfig(dataSource string) (*pgx.ConnConfig, error) {
	if dataSource == "" {
		dataSource = defaultDataSource
	}

	cfg, err := pgx.ParseConfig(dataSource)
	if err != nil {
		return nil, err
	}

	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = make(map[string]string)
	}

	// pgx doesn't support dbname so we emulate it
	if dbname, ok := cfg.RuntimeParams["dbname"]; ok {
		cfg.Database = dbname
		delete(cfg.RuntimeParams, "dbname")
	}

	// pgx doesn't support fallback_application_name so we emulate it
	// by checking if application_name is set and setting a default
	// value if not.
	if _, ok := cfg.RuntimeParams["application_name"]; !ok {
		cfg.RuntimeParams["application_name"] = defaultApplicationName
	}

	// Force PostgreSQL session timezone to UTC.
	// pgx doesn't support the PGTZ environment variable, we need to pass
	// that information in the configuration instead.
	tz := "UTC"
	if v, ok := os.LookupEnv("PGTZ"); ok && v != "UTC" && v != "utc" {
		log15.Warn("Ignoring PGTZ environment variable; using PGTZ=UTC.", "ignoredPGTZ", v)
	}
	// We set the environment variable to PGTZ to avoid bad surprises if and when
	// it will be supported by the driver.
	if err := os.Setenv("PGTZ", "UTC"); err != nil {
		return nil, errors.Wrap(err, "Error setting PGTZ=UTC")
	}
	cfg.RuntimeParams["timezone"] = tz

	// Ensure the TZ environment variable is set so that times are parsed correctly.
	if _, ok := os.LookupEnv("TZ"); !ok {
		log15.Warn("TZ environment variable not defined; using TZ=''.")
		if err := os.Setenv("TZ", ""); err != nil {
			return nil, errors.Wrap(err, "Error setting TZ=''")
		}
	}

	return cfg, nil
}

func openDBWithStartupWait(cfg *pgx.ConnConfig) (db *sql.DB, err error) {
	// Allow the DB to take up to 10s while it reports "pq: the database system is starting up".
	startupDeadline := time.Now().Add(startupTimeout)
	for {
		if time.Now().After(startupDeadline) {
			return nil, fmt.Errorf("database did not start up within %s (%v)", startupTimeout, err)
		}
		db, err = open(cfg)
		if err == nil {
			err = db.Ping()
		}
		if err != nil && isDatabaseLikelyStartingUp(err) {
			time.Sleep(startupTimeout / 10)
			continue
		}
		return db, err
	}
}

// isDatabaseLikelyStartingUp returns whether the err likely just means the PostgreSQL database is
// starting up, and it should not be treated as a fatal error during program initialization.
func isDatabaseLikelyStartingUp(err error) bool {
	msg := err.Error()
	if strings.Contains(msg, "the database system is starting up") {
		// Wait for DB to start up.
		return true
	}
	if strings.Contains(msg, "connection refused") || strings.Contains(msg, "failed to receive message") {
		// Wait for DB to start listening.
		return true
	}

	return false
}

var registerOnce sync.Once

// Open creates a new DB handle with the given schema by connecting to
// the database identified by dataSource (e.g., "dbname=mypgdb" or
// blank to use the PG* env vars).
//
// Open assumes that the database already exists.
func Open(dataSource string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(dataSource)
	if err != nil {
		return nil, err
	}

	return open(cfg)
}

func open(cfg *pgx.ConnConfig) (*sql.DB, error) {
	cfgKey := stdlib.RegisterConnConfig(cfg)

	registerOnce.Do(func() {
		sql.Register("postgres-proxy", sqlhooks.Wrap(stdlib.GetDefaultDriver(), &hook{}))
	})
	db, err := sql.Open("postgres-proxy", cfgKey)
	if err != nil {
		return nil, errors.Wrap(err, "postgresql open")
	}

	// Set max open and idle connections
	maxOpen, _ := strconv.Atoi(cfg.RuntimeParams["max_conns"])
	if maxOpen == 0 {
		maxOpen = 30
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxOpen)
	db.SetConnMaxLifetime(time.Minute)

	return db, nil
}

// Ping attempts to contact the database and returns a non-nil error upon failure. It is intended to
// be used by health checks.
func Ping(ctx context.Context) error { return Global.PingContext(ctx) }

type key int

const bulkInsertionKey key = iota

// BulkInsertion returns true if the bulkInsertionKey context value is true.
func BulkInsertion(ctx context.Context) bool {
	v, ok := ctx.Value(bulkInsertionKey).(bool)
	if !ok {
		return false
	}
	return v
}

// WithBulkInsertion sets the bulkInsertionKey context value.
func WithBulkInsertion(ctx context.Context, bulkInsertion bool) context.Context {
	return context.WithValue(ctx, bulkInsertionKey, bulkInsertion)
}

type hook struct{}

// postgresBulkInsertRowsPattern matches `($1, $2, $3), ($4, $5, $6), ...` which
// we use to cut out the row payloads from bulk insertion tracing data. We don't
// need all the parameter data for such requests, which are too big to fit into
// Jaeger spans. Note that we don't just capture `($1.*`, as we want queries with
// a trailing ON CONFLICT clause not to be semantically mangled in the log output.
var postgresBulkInsertRowsPattern = lazyregexp.New(`(\([$\d,\s]+\)[,\s]*)+`)

// postgresBulkInsertRowsReplacement replaces the all-placeholder rows matched
// by the pattern defined above.
var postgresBulkInsertRowsReplacement = []byte("(...) ")

// Before implements sqlhooks.Hooks
func (h *hook) Before(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	if BulkInsertion(ctx) {
		query = string(postgresBulkInsertRowsPattern.ReplaceAll([]byte(query), postgresBulkInsertRowsReplacement))
	}

	tr, ctx := trace.New(ctx, "sql", query,
		trace.Tag{Key: "span.kind", Value: "client"},
		trace.Tag{Key: "db.type", Value: "sql"},
	)

	if !BulkInsertion(ctx) {
		tr.LogFields(otlog.Lazy(func(fv otlog.Encoder) {
			for i, arg := range args {
				fv.EmitString(strconv.Itoa(i+1), fmt.Sprintf("%v", arg))
			}
		}))
	} else {
		tr.LogFields(otlog.Bool("bulk_insert", true), otlog.Int("num_args", len(args)))
	}

	return ctx, nil
}

// After implements sqlhooks.Hooks
func (h *hook) After(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	if tr := trace.TraceFromContext(ctx); tr != nil {
		tr.Finish()
	}
	return ctx, nil
}

// After implements sqlhooks.OnErroer
func (h *hook) OnError(ctx context.Context, err error, query string, args ...interface{}) error {
	if tr := trace.TraceFromContext(ctx); tr != nil {
		tr.SetError(err)
		tr.Finish()
	}
	return err
}

func registerPrometheusCollector(db *sql.DB, dbNameSuffix string) {
	c := prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "src",
			Subsystem: "pgsql" + strings.Replace(dbNameSuffix, "-", "_", -1),
			Name:      "open_connections",
			Help:      "Number of open connections to pgsql DB, as reported by pgsql.DB.Stats()",
		},
		func() float64 {
			s := db.Stats()
			return float64(s.OpenConnections)
		},
	)
	prometheus.MustRegister(c)
}

// configureConnectionPool sets reasonable sizes on the built in DB queue. By
// default the connection pool is unbounded, which leads to the error `pq:
// sorry too many clients already`.
func configureConnectionPool(db *sql.DB) {
	var err error
	maxOpen := 30
	if e := os.Getenv("SRC_PGSQL_MAX_OPEN"); e != "" {
		maxOpen, err = strconv.Atoi(e)
		if err != nil {
			log.Fatalf("SRC_PGSQL_MAX_OPEN is not an int: %s", e)
		}
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxOpen)
	db.SetConnMaxLifetime(time.Minute)
}
