package shared

import (
	"database/sql"
	"fmt"

	"github.com/sourcegraph/sourcegraph/internal/conf/conftypes"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
)

// InitDatabase initializes and returns a connection to the frontend database.
func InitDatabase() (*sql.DB, error) {
	conn, err := initDatabaseMemo.Init()
	return conn.(*sql.DB), err
}

var initDatabaseMemo = NewMemoizedConstructor(func() (interface{}, error) {
	postgresDSN := WatchServiceConnectionValue(func(serviceConnections conftypes.ServiceConnections) string {
		return serviceConnections.PostgresDSN
	})

	db, err := dbconn.New(postgresDSN, "_app")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to frontend database: %s", err)
	}

	return db, nil
})
