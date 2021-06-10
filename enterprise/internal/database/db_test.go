package database

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func equal(t testing.TB, name string, want, have interface{}) {
	t.Helper()
	if diff := cmp.Diff(want, have); diff != "" {
		t.Fatalf("%q: %s", name, diff)
	}
}
