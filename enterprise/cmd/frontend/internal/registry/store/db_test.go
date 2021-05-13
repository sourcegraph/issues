package store

import "github.com/sourcegraph/sourcegraph/internal/database/dbtesting"

func init() {
	dbtesting.DBNameSuffix = "registry"
}

func strptr(s string) *string {
	return &s
}
