package shared

import (
	"database/sql"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/conf/conftypes"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
)

// InitDatabase initializes and returns a connection to the frontend database.
func InitDatabase() (*sql.DB, error) {
	conn, err := initDatabaseMemo.Init()
	if err != nil {
		return nil, err
	}

	return conn.(*sql.DB), nil
}

var initDatabaseMemo = NewMemoizedConstructor(func() (interface{}, error) {
	postgresDSN := WatchServiceConnectionValue(func(serviceConnections conftypes.ServiceConnections) string {
		return serviceConnections.PostgresDSN
	})

	opts := dbconn.Opts{DSN: postgresDSN, DBName: "frontend", AppName: "worker"}
	if err := dbconn.SetupGlobalConnection(opts); err != nil {
		return nil, errors.Errorf("failed to connect to frontend database: %s", err)
	}

	return dbconn.Global, nil
})
