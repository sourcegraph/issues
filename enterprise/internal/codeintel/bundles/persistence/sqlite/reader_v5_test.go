package sqlite

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
)

const v5TestFile = "./testdata/lsif-go@5bc35c78.lsif.db"

func TestReadMetaV5(t *testing.T) {
	meta, err := testStore(t, v5TestFile).ReadMeta(context.Background())
	if err != nil {
		t.Fatalf("unexpected error reading meta: %s", err)
	}
	if meta.NumResultChunks != 4 {
		t.Errorf("unexpected numResultChunks. want=%d have=%d", 4, meta.NumResultChunks)
	}
}

func TestPathsWithPrefixV5(t *testing.T) {
	paths, err := testStore(t, v5TestFile).PathsWithPrefix(context.Background(), "internal/")
	if err != nil {
		t.Fatalf("unexpected error fetching paths with prefix: %s", err)
	}

	expectedPaths := []string{
		"internal/gomod/module.go",
		"internal/index/helper.go",
		"internal/index/indexer.go",
		"internal/index/types.go",
	}
	if diff := cmp.Diff(expectedPaths, paths); diff != "" {
		t.Errorf("unexpected paths (-want +got):\n%s", diff)
	}
}

func TestReadDocumentV5(t *testing.T) {
	data, exists, err := testStore(t, v5TestFile).ReadDocument(context.Background(), "protocol/writer.go")
	if err != nil {
		t.Fatalf("unexpected error reading document: %s", err)
	}
	if !exists {
		t.Errorf("expected document to exist")
	}

	expectedRange := types.RangeData{
		StartLine:          140,
		StartCharacter:     17,
		EndLine:            140,
		EndCharacter:       39,
		DefinitionResultID: types.ID("3457"),
		ReferenceResultID:  types.ID("15737"),
		HoverResultID:      types.ID("3463"),
		MonikerIDs:         []types.ID{types.ID("3460")},
	}
	if diff := cmp.Diff(expectedRange, data.Ranges[types.ID("3454")]); diff != "" {
		t.Errorf("unexpected range data (-want +got):\n%s", diff)
	}

	expectedHoverData := "```go\nfunc (*Writer).EmitPackageInformation(packageName string, scheme string, version string) (string, error)\n```"
	if diff := cmp.Diff(expectedHoverData, data.HoverResults[types.ID("3463")]); diff != "" {
		t.Errorf("unexpected hover data (-want +got):\n%s", diff)
	}

	expectedMoniker := types.MonikerData{
		Kind:                 "export",
		Scheme:               "gomod",
		Identifier:           "github.com/sourcegraph/lsif-go/protocol:Writer.EmitPackageInformation",
		PackageInformationID: types.ID("37"),
	}
	if diff := cmp.Diff(expectedMoniker, data.Monikers[types.ID("3460")]); diff != "" {
		t.Errorf("unexpected moniker data (-want +got):\n%s", diff)
	}

	expectedPackageInformation := types.PackageInformationData{
		Name:    "github.com/sourcegraph/lsif-go",
		Version: "v0.8.0",
	}
	if diff := cmp.Diff(expectedPackageInformation, data.PackageInformation[types.ID("37")]); diff != "" {
		t.Errorf("unexpected package information data (-want +got):\n%s", diff)
	}
}

func TestReadResultChunkV5(t *testing.T) {
	data, exists, err := testStore(t, v5TestFile).ReadResultChunk(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error reading result chunk: %s", err)
	}
	if !exists {
		t.Errorf("expected result chunk to exist")
	}

	if path := data.DocumentPaths[types.ID("356")]; path != "protocol/writer.go" {
		t.Errorf("unexpected document path. want=%s have=%s", "protocol/writer.go", path)
	}

	expectedDocumentRanges := []types.DocumentIDRangeID{
		{DocumentID: "4032", RangeID: "5606"},
		{DocumentID: "4032", RangeID: "11714"},
		{DocumentID: "4032", RangeID: "11947"},
	}

	if diff := cmp.Diff(expectedDocumentRanges, data.DocumentIDRangeIDs[types.ID("16772")]); diff != "" {
		t.Errorf("unexpected document ranges (-want +got):\n%s", diff)
	}
}

func TestReadDefinitionsV5(t *testing.T) {
	definitions, totalCount, err := testStore(t, v5TestFile).ReadDefinitions(context.Background(), "gomod", "github.com/sourcegraph/lsif-go/protocol:Vertex", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error getting definitions: %s", err)
	}
	if totalCount != 1 {
		t.Errorf("unexpected total count. want=%d have=%d", 1, totalCount)
	}

	expectedDefinitions := []types.Location{
		{URI: "protocol/protocol.go", StartLine: 34, StartCharacter: 5, EndLine: 34, EndCharacter: 11},
	}
	if diff := cmp.Diff(expectedDefinitions, definitions); diff != "" {
		t.Errorf("unexpected definitions (-want +got):\n%s", diff)
	}
}

func TestReadReferencesV5(t *testing.T) {
	references, totalCount, err := testStore(t, v5TestFile).ReadReferences(context.Background(), "gomod", "golang.org/x/tools/go/packages:Package", 3, 4)
	if err != nil {
		t.Fatalf("unexpected error getting references: %s", err)
	}
	if totalCount != 25 {
		t.Errorf("unexpected total count. want=%d have=%d", 25, totalCount)
	}

	expectedReferences := []types.Location{
		{URI: "internal/index/indexer.go", StartLine: 275, StartCharacter: 48, EndLine: 275, EndCharacter: 55},
		{URI: "internal/index/indexer.go", StartLine: 486, StartCharacter: 66, EndLine: 486, EndCharacter: 73},
		{URI: "internal/index/indexer.go", StartLine: 647, StartCharacter: 86, EndLine: 647, EndCharacter: 93},
		{URI: "internal/index/indexer.go", StartLine: 131, StartCharacter: 41, EndLine: 131, EndCharacter: 48},
	}
	if diff := cmp.Diff(expectedReferences, references); diff != "" {
		t.Errorf("unexpected references (-want +got):\n%s", diff)
	}
}
