package database

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client_types"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/cache"
	sqlitereader "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/sqlite"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/sqliteutil"
)

func init() {
	sqliteutil.SetLocalLibpath()
	sqliteutil.MustRegisterSqlite3WithPcre()
}

func TestDatabaseExists(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"cmd/lsif-go/main.go", true},
		{"internal/index/indexer.go", true},
		{"missing.go", false},
	}

	db := openTestDatabase(t)
	for _, testCase := range testCases {
		if exists, err := db.Exists(context.Background(), testCase.path); err != nil {
			t.Fatalf("unexpected error %s", err)
		} else if exists != testCase.expected {
			t.Errorf("unexpected exists result for %s. want=%v have=%v", testCase.path, testCase.expected, exists)
		}
	}
}

func TestDatabaseRanges(t *testing.T) {
	//   20: // NewWriter creates a new Writer.
	//   21: func NewWriter(w io.Writer, addContents bool) *Writer {
	// > 22:     return &Writer{
	// > 23:         w:           w,
	// > 24:         addContents: addContents,
	//   25:     }
	//   26: }
	//   27:
	//   28: func (w *Writer) NumElements() int {
	//   29:     return w.numElements
	//   30: }

	db := openTestDatabase(t)
	if actual, err := db.Ranges(context.Background(), "protocol/writer.go", 21, 24); err != nil {
		t.Fatalf("unexpected error %s", err)
	} else {
		expected := []bundles.CodeIntelligenceRange{
			{
				Range: newRange(21, 9, 21, 15),
				Definitions: []bundles.Location{
					{Path: "protocol/writer.go", Range: newRange(12, 5, 12, 11)},
				},
				References: []bundles.Location{
					{Path: "protocol/writer.go", Range: newRange(12, 5, 12, 11)},
					{Path: "protocol/writer.go", Range: newRange(20, 47, 20, 53)},
					{Path: "protocol/writer.go", Range: newRange(21, 9, 21, 15)},
					{Path: "protocol/writer.go", Range: newRange(27, 9, 27, 15)},
					{Path: "protocol/writer.go", Range: newRange(31, 9, 31, 15)},
					{Path: "protocol/writer.go", Range: newRange(36, 9, 36, 15)},
					{Path: "protocol/writer.go", Range: newRange(41, 9, 41, 15)},
					{Path: "protocol/writer.go", Range: newRange(46, 9, 46, 15)},
					{Path: "protocol/writer.go", Range: newRange(51, 9, 51, 15)},
					{Path: "protocol/writer.go", Range: newRange(56, 9, 56, 15)},
					{Path: "protocol/writer.go", Range: newRange(61, 9, 61, 15)},
					{Path: "protocol/writer.go", Range: newRange(75, 9, 75, 15)},
					{Path: "protocol/writer.go", Range: newRange(80, 9, 80, 15)},
					{Path: "protocol/writer.go", Range: newRange(85, 9, 85, 15)},
					{Path: "protocol/writer.go", Range: newRange(90, 9, 90, 15)},
					{Path: "protocol/writer.go", Range: newRange(95, 9, 95, 15)},
					{Path: "protocol/writer.go", Range: newRange(100, 9, 100, 15)},
					{Path: "protocol/writer.go", Range: newRange(105, 9, 105, 15)},
					{Path: "protocol/writer.go", Range: newRange(110, 9, 110, 15)},
					{Path: "protocol/writer.go", Range: newRange(115, 9, 115, 15)},
					{Path: "protocol/writer.go", Range: newRange(120, 9, 120, 15)},
					{Path: "protocol/writer.go", Range: newRange(125, 9, 125, 15)},
					{Path: "protocol/writer.go", Range: newRange(130, 9, 130, 15)},
					{Path: "protocol/writer.go", Range: newRange(135, 9, 135, 15)},
					{Path: "protocol/writer.go", Range: newRange(140, 9, 140, 15)},
					{Path: "protocol/writer.go", Range: newRange(145, 9, 145, 15)},
					{Path: "protocol/writer.go", Range: newRange(150, 9, 150, 15)},
					{Path: "protocol/writer.go", Range: newRange(155, 9, 155, 15)},
				},
				HoverText: "```go\ntype Writer struct\n```\n\n---\n\nWriter emits vertices and edges to the underlying writer. This struct will guarantee that unique identifiers are generated for each element.\n\n---\n\n```go\nstruct {\n    w Writer\n    addContents bool\n    id int\n    numElements int\n}\n```",
			},
			{
				Range: newRange(22, 2, 22, 3),
				Definitions: []bundles.Location{
					{Path: "protocol/writer.go", Range: newRange(13, 1, 13, 2)},
				},
				References: []bundles.Location{
					{Path: "protocol/writer.go", Range: newRange(13, 1, 13, 2)},
					{Path: "protocol/writer.go", Range: newRange(22, 2, 22, 3)},
					{Path: "protocol/writer.go", Range: newRange(38, 26, 38, 27)},
				},
				HoverText: "```go\nstruct field w io.Writer\n```",
			},
			{
				Range: newRange(22, 15, 22, 16),
				Definitions: []bundles.Location{
					{Path: "protocol/writer.go", Range: newRange(20, 15, 20, 16)},
				},
				References: []bundles.Location{
					{Path: "protocol/writer.go", Range: newRange(20, 15, 20, 16)},
					{Path: "protocol/writer.go", Range: newRange(22, 15, 22, 16)},
				},
				HoverText: "```go\nvar w Writer\n```",
			},
			{
				Range: newRange(23, 2, 23, 13),
				Definitions: []bundles.Location{
					{Path: "protocol/writer.go", Range: newRange(14, 1, 14, 12)},
				},
				References: []bundles.Location{
					{Path: "protocol/writer.go", Range: newRange(14, 1, 14, 12)},
					{Path: "protocol/writer.go", Range: newRange(23, 2, 23, 13)},
					{Path: "protocol/writer.go", Range: newRange(63, 6, 63, 17)},
				},
				HoverText: "```go\nstruct field addContents bool\n```",
			},
			{
				Range: newRange(23, 15, 23, 26),
				Definitions: []bundles.Location{
					{Path: "protocol/writer.go", Range: newRange(20, 28, 20, 39)},
				},
				References: []bundles.Location{
					{Path: "protocol/writer.go", Range: newRange(20, 28, 20, 39)},
					{Path: "protocol/writer.go", Range: newRange(23, 15, 23, 26)},
				},
				HoverText: "```go\nvar addContents bool\n```",
			},
		}

		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("unexpected definitions locations (-want +got):\n%s", diff)
		}
	}
}

func TestDatabaseDefinitions(t *testing.T) {
	// `\ts, err := indexer.Index()` -> `\t Index() (*Stats, error)`
	//                      ^^^^^           ^^^^^

	db := openTestDatabase(t)
	if actual, err := db.Definitions(context.Background(), "cmd/lsif-go/main.go", 110, 22); err != nil {
		t.Fatalf("unexpected error %s", err)
	} else {
		expected := []bundles.Location{
			{
				Path:  "internal/index/indexer.go",
				Range: newRange(20, 1, 20, 6),
			},
		}

		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("unexpected definitions locations (-want +got):\n%s", diff)
		}
	}
}

func TestDatabaseReferences(t *testing.T) {
	// `func (w *Writer) EmitRange(start, end Pos) (string, error) {`
	//                   ^^^^^^^^^
	//
	// -> `\t\trangeID, err := i.w.EmitRange(lspRange(ipos, ident.Name, isQuotedPkgName))`
	//                             ^^^^^^^^^
	//
	// -> `\t\t\trangeID, err = i.w.EmitRange(lspRange(ipos, ident.Name, false))`
	//                              ^^^^^^^^^

	db := openTestDatabase(t)
	if actual, err := db.References(context.Background(), "protocol/writer.go", 85, 20); err != nil {
		t.Fatalf("unexpected error %s", err)
	} else {
		expected := []bundles.Location{
			{
				Path:  "internal/index/indexer.go",
				Range: newRange(380, 22, 380, 31),
			},
			{
				Path:  "internal/index/indexer.go",
				Range: newRange(529, 22, 529, 31),
			},
			{
				Path:  "protocol/writer.go",
				Range: newRange(85, 17, 85, 26),
			},
		}

		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("unexpected reference locations (-want +got):\n%s", diff)
		}
	}
}

func TestDatabaseHover(t *testing.T) {
	// `\tcontents, err := findContents(pkgs, p, f, obj)`
	//                     ^^^^^^^^^^^^

	db := openTestDatabase(t)
	if actualText, actualRange, exists, err := db.Hover(context.Background(), "internal/index/indexer.go", 628, 20); err != nil {
		t.Fatalf("unexpected error %s", err)
	} else if !exists {
		t.Errorf("no hover found")
	} else {
		docstring := "findContents returns contents used as hover info for given object."
		signature := "func findContents(pkgs []*Package, p *Package, f *File, obj Object) ([]MarkedString, error)"
		expectedText := "```go\n" + signature + "\n```\n\n---\n\n" + docstring
		expectedRange := newRange(628, 18, 628, 30)

		if actualText != expectedText {
			t.Errorf("unexpected hover text. want=%s have=%s", expectedText, actualText)
		}

		if diff := cmp.Diff(expectedRange, actualRange); diff != "" {
			t.Errorf("unexpected hover range (-want +got):\n%s", diff)
		}
	}
}

func TestDatabaseMonikersByPosition(t *testing.T) {
	// `func NewMetaData(id, root string, info ToolInfo) *MetaData {`
	//       ^^^^^^^^^^^

	db := openTestDatabase(t)
	if actual, err := db.MonikersByPosition(context.Background(), "protocol/protocol.go", 92, 10); err != nil {
		t.Fatalf("unexpected error %s", err)
	} else {
		expected := [][]bundles.MonikerData{
			{
				{
					Kind:                 "export",
					Scheme:               "gomod",
					Identifier:           "github.com/sourcegraph/lsif-go/protocol:NewMetaData",
					PackageInformationID: "213",
				},
			},
		}

		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("unexpected moniker result (-want +got):\n%s", diff)
		}
	}
}

func TestDatabaseMonikerResults(t *testing.T) {
	edgeLocations := []bundles.Location{
		{
			Path:  "protocol/protocol.go",
			Range: newRange(600, 1, 600, 5),
		},
		{
			Path:  "protocol/protocol.go",
			Range: newRange(644, 1, 644, 5),
		},
		{
			Path:  "protocol/protocol.go",
			Range: newRange(507, 1, 507, 5),
		},
		{
			Path:  "protocol/protocol.go",
			Range: newRange(553, 1, 553, 5),
		},
		{
			Path:  "protocol/protocol.go",
			Range: newRange(462, 1, 462, 5),
		},
		{
			Path:  "protocol/protocol.go",
			Range: newRange(484, 1, 484, 5),
		},
		{
			Path:  "protocol/protocol.go",
			Range: newRange(410, 5, 410, 9),
		},
		{
			Path:  "protocol/protocol.go",
			Range: newRange(622, 1, 622, 5),
		},
		{
			Path:  "protocol/protocol.go",
			Range: newRange(440, 1, 440, 5),
		},
		{
			Path:  "protocol/protocol.go",
			Range: newRange(530, 1, 530, 5),
		},
	}

	markdownLocations := []bundles.Location{
		{
			Path:  "internal/index/helper.go",
			Range: newRange(78, 6, 78, 16),
		},
	}

	testCases := []struct {
		tableName          string
		scheme             string
		identifier         string
		skip               int
		take               int
		expectedLocations  []bundles.Location
		expectedTotalCount int
	}{
		{"definitions", "gomod", "github.com/sourcegraph/lsif-go/protocol:Edge", 0, 100, edgeLocations, 10},
		{"definitions", "gomod", "github.com/sourcegraph/lsif-go/protocol:Edge", 3, 4, edgeLocations[3:7], 10},
		{"references", "gomod", "github.com/slimsag/godocmd:ToMarkdown", 0, 100, markdownLocations, 1},
	}

	db := openTestDatabase(t)
	for i, testCase := range testCases {
		if actual, totalCount, err := db.MonikerResults(context.Background(), testCase.tableName, testCase.scheme, testCase.identifier, testCase.skip, testCase.take); err != nil {
			t.Fatalf("unexpected error for test case #%d: %s", i, err)
		} else {
			if totalCount != testCase.expectedTotalCount {
				t.Errorf("unexpected moniker result total count for test case #%d. want=%d have=%d", i, testCase.expectedTotalCount, totalCount)
			}

			if diff := cmp.Diff(testCase.expectedLocations, actual); diff != "" {
				t.Errorf("unexpected moniker result locations for test case #%d (-want +got):\n%s", i, diff)
			}
		}
	}
}

func TestDatabasePackageInformation(t *testing.T) {
	db := openTestDatabase(t)
	if actual, exists, err := db.PackageInformation(context.Background(), "protocol/protocol.go", "213"); err != nil {
		t.Fatalf("unexpected error %s", err)
	} else if !exists {
		t.Errorf("no package information")
	} else {
		expected := bundles.PackageInformationData{
			Name:    "github.com/sourcegraph/lsif-go",
			Version: "v0.0.0-ad3507cbeb18",
		}

		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("unexpected package information (-want +got):\n%s", diff)
		}
	}
}

func openTestDatabase(t *testing.T) Database {
	filename := copyFile(t, "../../../../internal/codeintel/bundles/persistence/sqlite/testdata/lsif-go@ad3507cb.lsif.db")

	cache, err := cache.NewDataCache(10)
	if err != nil {
		t.Fatalf("unexpected error creating cache: %s", err)
	}

	// TODO(efritz) - rewrite test not to require actual store
	store, err := sqlitereader.OpenStore(context.Background(), filename, cache)
	if err != nil {
		t.Fatalf("unexpected error creating store: %s", err)
	}

	db, err := OpenDatabase(context.Background(), filename, store)
	if err != nil {
		t.Fatalf("unexpected error opening database: %s", err)
	}
	t.Cleanup(func() { _ = db.Close })

	// Wrap in observed, as that's how it's used in production
	return NewObserved(db, filename, &observation.TestContext)
}

func copyFile(t *testing.T, source string) string {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("unexpected error creating temp dir: %s", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	input, err := ioutil.ReadFile(source)
	if err != nil {
		t.Fatalf("unexpected error reading file: %s", err)
	}

	dest := filepath.Join(tempDir, "test.sqlite")
	if err := ioutil.WriteFile(dest, input, os.ModePerm); err != nil {
		t.Fatalf("unexpected error writing file: %s", err)
	}

	return dest
}
