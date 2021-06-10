package lsifstore

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

func TestDatabaseDefinitions(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	populateTestStore(t, db)
	store := NewStore(db, &observation.TestContext)

	// `\ts, err := indexer.Index()` -> `\t Index() (*Stats, error)`
	//                      ^^^^^           ^^^^^

	if actual, _, err := store.Definitions(context.Background(), testBundleID, "cmd/lsif-go/main.go", 110, 22, 5, 0); err != nil {
		t.Fatalf("unexpected error %s", err)
	} else {
		expected := []Location{
			{DumpID: testBundleID, Path: "internal/index/indexer.go", Range: newRange(20, 1, 20, 6)},
		}

		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("unexpected definitions locations (-want +got):\n%s", diff)
		}
	}
}

func TestDatabaseReferences(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	populateTestStore(t, db)
	store := NewStore(db, &observation.TestContext)

	// `func (w *Writer) EmitRange(start, end Pos) (string, error) {`
	//                   ^^^^^^^^^
	//
	// -> `\t\trangeID, err := i.w.EmitRange(lspRange(ipos, ident.Name, isQuotedPkgName))`
	//                             ^^^^^^^^^
	//
	// -> `\t\t\trangeID, err = i.w.EmitRange(lspRange(ipos, ident.Name, false))`
	//                              ^^^^^^^^^

	expected := []Location{
		{DumpID: testBundleID, Path: "internal/index/indexer.go", Range: newRange(380, 22, 380, 31)},
		{DumpID: testBundleID, Path: "internal/index/indexer.go", Range: newRange(529, 22, 529, 31)},
		{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(85, 17, 85, 26)},
	}

	testCases := []struct {
		limit    int
		offset   int
		expected []Location
	}{
		{5, 0, expected},
		{2, 0, expected[:2]},
		{2, 1, expected[1:]},
		{5, 5, expected[:0]},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("i=%d", i), func(t *testing.T) {
			if actual, totalCount, err := store.References(context.Background(), testBundleID, "protocol/writer.go", 85, 20, testCase.limit, testCase.offset); err != nil {
				t.Fatalf("unexpected error %s", err)
			} else {
				if totalCount != 3 {
					t.Errorf("unexpected count. want=%d have=%d", 3, totalCount)
				}

				if diff := cmp.Diff(testCase.expected, actual); diff != "" {
					t.Errorf("unexpected reference locations (-want +got):\n%s", diff)
				}
			}
		})
	}
}
