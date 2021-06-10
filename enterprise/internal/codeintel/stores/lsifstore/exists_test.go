package lsifstore

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

func TestDatabaseExists(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	populateTestStore(t, db)
	store := NewStore(db, &observation.TestContext)

	testCases := []struct {
		path     string
		expected bool
	}{
		{"cmd/lsif-go/main.go", true},
		{"internal/index/indexer.go", true},
		{"missing.go", false},
	}

	for _, testCase := range testCases {
		if exists, err := store.Exists(context.Background(), testBundleID, testCase.path); err != nil {
			t.Fatalf("unexpected error %s", err)
		} else if exists != testCase.expected {
			t.Errorf("unexpected exists result for %s. want=%v have=%v", testCase.path, testCase.expected, exists)
		}
	}
}
