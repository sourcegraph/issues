package lsifstore

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

func TestDatabaseRanges(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	populateTestStore(t, db)
	store := NewStore(db, &observation.TestContext)

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

	if actual, err := store.Ranges(context.Background(), testBundleID, "protocol/writer.go", 21, 24); err != nil {
		t.Fatalf("unexpected error %s", err)
	} else {
		expected := []CodeIntelligenceRange{
			{
				Range: newRange(21, 9, 21, 15),
				Definitions: []Location{
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(12, 5, 12, 11)},
				},
				References: []Location{
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(12, 5, 12, 11)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(20, 47, 20, 53)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(21, 9, 21, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(27, 9, 27, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(31, 9, 31, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(36, 9, 36, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(41, 9, 41, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(46, 9, 46, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(51, 9, 51, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(56, 9, 56, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(61, 9, 61, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(75, 9, 75, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(80, 9, 80, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(85, 9, 85, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(90, 9, 90, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(95, 9, 95, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(100, 9, 100, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(105, 9, 105, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(110, 9, 110, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(115, 9, 115, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(120, 9, 120, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(125, 9, 125, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(130, 9, 130, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(135, 9, 135, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(140, 9, 140, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(145, 9, 145, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(150, 9, 150, 15)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(155, 9, 155, 15)},
				},
				HoverText: "```go\ntype Writer struct\n```\n\n---\n\nWriter emits vertices and edges to the underlying writer. This struct will guarantee that unique identifiers are generated for each element.\n\n---\n\n```go\nstruct {\n    w Writer\n    addContents bool\n    id int\n    numElements int\n}\n```",
			},
			{
				Range: newRange(22, 2, 22, 3),
				Definitions: []Location{
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(13, 1, 13, 2)},
				},
				References: []Location{
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(13, 1, 13, 2)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(22, 2, 22, 3)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(38, 26, 38, 27)},
				},
				HoverText: "```go\nstruct field w io.Writer\n```",
			},
			{
				Range: newRange(22, 15, 22, 16),
				Definitions: []Location{
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(20, 15, 20, 16)},
				},
				References: []Location{
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(20, 15, 20, 16)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(22, 15, 22, 16)},
				},
				HoverText: "```go\nvar w Writer\n```",
			},
			{
				Range: newRange(23, 2, 23, 13),
				Definitions: []Location{
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(14, 1, 14, 12)},
				},
				References: []Location{
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(14, 1, 14, 12)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(23, 2, 23, 13)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(63, 6, 63, 17)},
				},
				HoverText: "```go\nstruct field addContents bool\n```",
			},
			{
				Range: newRange(23, 15, 23, 26),
				Definitions: []Location{
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(20, 28, 20, 39)},
				},
				References: []Location{
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(20, 28, 20, 39)},
					{DumpID: testBundleID, Path: "protocol/writer.go", Range: newRange(23, 15, 23, 26)},
				},
				HoverText: "```go\nvar addContents bool\n```",
			},
		}

		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("unexpected definitions locations (-want +got):\n%s", diff)
		}
	}
}
