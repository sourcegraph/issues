package db

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func init() {
	dbtesting.DBNameSuffix = "enterprisedb"
}

func equal(t testing.TB, name string, want, have interface{}) {
	t.Helper()
	if diff := cmp.Diff(want, have); diff != "" {
		t.Fatalf("%q: %s", name, diff)
	}
}
