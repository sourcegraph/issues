package postgres

import "github.com/sourcegraph/sourcegraph/internal/sqliteutil"

func init() {
	sqliteutil.SetLocalLibpath()
	sqliteutil.MustRegisterSqlite3WithPcre()
}
