package dbconn

import (
	"context"
	"log"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Restricted is the "sg_service" DB connection.
	// Only use this after a call to SetupRestrictedConnection.
	Restricted *pgxpool.Pool
)

// SetupRestrictedConnection connects to the given data source, ensures reduced
// privileges, and stores the handle globally.
//
// Each connection in the pool will adopt the "sg_service" role upon being checked
// out. This role has row security policies applied to it so that various authn/z
// checks are handled at the Postgres level instead of application code. This lowers
// the burden on developers, since you only have to opt-in to using the connection
// vs. invoking specific functions at specific times.
//
// The restricted connection is configured after the base connection, so we don't
// need to handle the various liveness checks (e.g. is the database up). If we
// reached this point, it is.
func SetupRestrictedConnection(opts Opts) (err error) {
	Restricted, err = NewRestricted(opts)
	return err
}

// NewRestricted connects to the given data source, sets up the connection lifecycle
// hooks, and returns the handle to the pool.
func NewRestricted(opts Opts) (*pgxpool.Pool, error) {
	config, err := buildPoolConfig(opts.DSN, opts.AppName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to build pool config")
	}

	// When each connection is checked out of the pool, we ensure that the service
	// role is assumed automatically and transparently.
	config.BeforeAcquire = func(ctx context.Context, c *pgx.Conn) bool {
		if _, err := c.Exec(ctx, "SET SESSION AUTHORIZATION 'sg_service';"); err != nil {
			log.Println(errors.Wrap(err, "unable to assume sg_service role"))
			return false
		}
		return true
	}

	// Connect to the database and register Prometheus metrics for observability.
	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect pgx pool")
	}
	prometheus.MustRegister(newMetricsRestrictedCollector(pool, opts.DBName, opts.AppName))
	return pool, nil
}

// buildPoolConfig takes either a Postgres connection string or connection URI,
// parses it, and returns a pool config with additional parameters.
func buildPoolConfig(dataSource, app string) (*pgxpool.Config, error) {
	config, err := buildConfig(dataSource, app)
	if err != nil {
		return nil, err
	}
	poolConfig, err := pgxpool.ParseConfig(dataSource)
	if err != nil {
		return nil, err
	}
	poolConfig.ConnConfig = config
	poolConfig.MinConns = int32(0)
	poolConfig.MaxConns = int32(Global.Stats().MaxOpenConnections)
	poolConfig.MaxConnIdleTime = time.Minute
	poolConfig.MaxConnLifetime = time.Hour
	return poolConfig, nil
}
