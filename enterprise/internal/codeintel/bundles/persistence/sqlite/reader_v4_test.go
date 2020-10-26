package sqlite

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
)

const v4TestFile = "./testdata/lsif-go@70ce6dad.lsif.db"

func TestReadMetaV4(t *testing.T) {
	meta, err := testStore(t, v4TestFile).ReadMeta(context.Background())
	if err != nil {
		t.Fatalf("unexpected error reading meta: %s", err)
	}
	if meta.NumResultChunks != 4 {
		t.Errorf("unexpected numResultChunks. want=%d have=%d", 4, meta.NumResultChunks)
	}
}

func TestPathsWithPrefixV4(t *testing.T) {
	paths, err := testStore(t, v4TestFile).PathsWithPrefix(context.Background(), "internal/")
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
func TestReadDocumentV4(t *testing.T) {
	data, exists, err := testStore(t, v4TestFile).ReadDocument(context.Background(), "protocol/writer.go")
	if err != nil {
		t.Fatalf("unexpected error reading document: %s", err)
	}
	if !exists {
		t.Errorf("expected document to exist")
	}

	expectedRange := types.RangeData{
		StartLine:          95,
		StartCharacter:     17,
		EndLine:            95,
		EndCharacter:       37,
		DefinitionResultID: types.ID("3521"),
		ReferenceResultID:  types.ID("17407"),
		HoverResultID:      types.ID("3527"),
		MonikerIDs:         []types.ID{types.ID("3524")},
	}
	if diff := cmp.Diff(expectedRange, data.Ranges[types.ID("3518")]); diff != "" {
		t.Errorf("unexpected range data (-want +got):\n%s", diff)
	}

	expectedHoverData := "```go\nfunc (*Writer).EmitContains(outV string, inVs []string) (string, error)\n```"
	if diff := cmp.Diff(expectedHoverData, data.HoverResults[types.ID("2946")]); diff != "" {
		t.Errorf("unexpected hover data (-want +got):\n%s", diff)
	}

	expectedMoniker := types.MonikerData{
		Kind:                 "export",
		Scheme:               "gomod",
		Identifier:           "github.com/sourcegraph/lsif-go/protocol:NewEvent",
		PackageInformationID: types.ID("133"),
	}
	if diff := cmp.Diff(expectedMoniker, data.Monikers[types.ID("1801")]); diff != "" {
		t.Errorf("unexpected moniker data (-want +got):\n%s", diff)
	}

	expectedPackageInformation := types.PackageInformationData{
		Name:    "github.com/sourcegraph/lsif-go",
		Version: "v0.8.0-70ce6dad37a4",
	}
	if diff := cmp.Diff(expectedPackageInformation, data.PackageInformation[types.ID("133")]); diff != "" {
		t.Errorf("unexpected package information data (-want +got):\n%s", diff)
	}
}

func TestReadResultChunkV4(t *testing.T) {
	data, exists, err := testStore(t, v4TestFile).ReadResultChunk(context.Background(), 2)
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
		{DocumentID: "353", RangeID: "2310"},
		{DocumentID: "353", RangeID: "7831"},
		{DocumentID: "353", RangeID: "8047"},
	}

	if diff := cmp.Diff(expectedDocumentRanges, data.DocumentIDRangeIDs[types.ID("16887")]); diff != "" {
		t.Errorf("unexpected document ranges (-want +got):\n%s", diff)
	}
}

func TestReadDefinitionsV4(t *testing.T) {
	definitions, totalCount, err := testStore(t, v4TestFile).ReadDefinitions(context.Background(), "gomod", "github.com/sourcegraph/lsif-go/protocol:Vertex", 0, 0)
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

func TestReadReferencesV4(t *testing.T) {
	references, totalCount, err := testStore(t, v4TestFile).ReadReferences(context.Background(), "gomod", "golang.org/x/tools/go/packages:Package", 3, 4)
	if err != nil {
		t.Fatalf("unexpected error getting references: %s", err)
	}
	if totalCount != 25 {
		t.Errorf("unexpected total count. want=%d have=%d", 25, totalCount)
	}

	expectedReferences := []types.Location{
		{URI: "internal/index/indexer.go", StartLine: 275, StartCharacter: 69, EndLine: 275, EndCharacter: 76},
		{URI: "internal/index/indexer.go", StartLine: 302, StartCharacter: 48, EndLine: 302, EndCharacter: 55},
		{URI: "internal/index/indexer.go", StartLine: 302, StartCharacter: 69, EndLine: 302, EndCharacter: 76},
		{URI: "internal/index/indexer.go", StartLine: 486, StartCharacter: 45, EndLine: 486, EndCharacter: 52},
	}
	if diff := cmp.Diff(expectedReferences, references); diff != "" {
		t.Errorf("unexpected references (-want +got):\n%s", diff)
	}
}
