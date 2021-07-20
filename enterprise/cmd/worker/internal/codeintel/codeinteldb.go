package codeintel

import (
	"database/sql"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/cmd/worker/shared"
	"github.com/sourcegraph/sourcegraph/internal/conf/conftypes"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
)

// InitCodeIntelDatabase initializes and returns a connection to the codeintel db.
func InitCodeIntelDatabase() (*sql.DB, error) {
	conn, err := initCodeIntelDatabaseMemo.Init()
	if err != nil {
		return nil, err
	}

	return conn.(*sql.DB), err
}

var initCodeIntelDatabaseMemo = shared.NewMemoizedConstructor(func() (interface{}, error) {
	postgresDSN := shared.WatchServiceConnectionValue(func(serviceConnections conftypes.ServiceConnections) string {
		return serviceConnections.CodeIntelPostgresDSN
	})

	db, err := dbconn.New(dbconn.Opts{DSN: postgresDSN, DBName: "codeintel", AppName: "worker"})
	if err != nil {
		return nil, errors.Errorf("failed to connect to codeintel database: %s", err)
	}

	if err := dbconn.MigrateDB(db, dbconn.CodeIntel); err != nil {
		return nil, errors.Errorf("failed to perform codeintel database migration: %s", err)
	}

	return db, nil
})
