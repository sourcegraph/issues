package sqlite

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
)

const v0TestFile = "./testdata/lsif-go@ad3507cb.lsif.db"

func TestReadMetaV0(t *testing.T) {
	meta, err := testStore(t, v0TestFile).ReadMeta(context.Background())
	if err != nil {
		t.Fatalf("unexpected error reading meta: %s", err)
	}
	if meta.NumResultChunks != 4 {
		t.Errorf("unexpected numResultChunks. want=%d have=%d", 4, meta.NumResultChunks)
	}
}

func TestPathsWithPrefixV0(t *testing.T) {
	paths, err := testStore(t, v0TestFile).PathsWithPrefix(context.Background(), "internal/")
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

func TestReadDocumentV0(t *testing.T) {
	data, exists, err := testStore(t, v0TestFile).ReadDocument(context.Background(), "protocol/writer.go")
	if err != nil {
		t.Fatalf("unexpected error reading document: %s", err)
	}
	if !exists {
		t.Errorf("expected document to exist")
	}

	expectedRange := types.RangeData{
		StartLine:          145,
		StartCharacter:     17,
		EndLine:            145,
		EndCharacter:       28,
		DefinitionResultID: types.ID("2873"),
		ReferenceResultID:  types.ID("16518"),
		HoverResultID:      types.ID("2879"),
		MonikerIDs:         []types.ID{types.ID("2876")},
	}
	if diff := cmp.Diff(expectedRange, data.Ranges[types.ID("2870")]); diff != "" {
		t.Errorf("unexpected range data (-want +got):\n%s", diff)
	}

	expectedHoverData := "```go\n" + `func (*Writer).EmitMoniker(kind string, scheme string, identifier string) (string, error)` + "\n```"
	if diff := cmp.Diff(expectedHoverData, data.HoverResults[types.ID("2879")]); diff != "" {
		t.Errorf("unexpected hover data (-want +got):\n%s", diff)
	}

	expectedMoniker := types.MonikerData{
		Kind:                 "export",
		Scheme:               "gomod",
		Identifier:           "github.com/sourcegraph/lsif-go/protocol:EmitMoniker",
		PackageInformationID: types.ID("213"),
	}
	if diff := cmp.Diff(expectedMoniker, data.Monikers[types.ID("2876")]); diff != "" {
		t.Errorf("unexpected moniker data (-want +got):\n%s", diff)
	}

	expectedPackageInformation := types.PackageInformationData{
		Name:    "github.com/sourcegraph/lsif-go",
		Version: "v0.0.0-ad3507cbeb18",
	}
	if diff := cmp.Diff(expectedPackageInformation, data.PackageInformation[types.ID("213")]); diff != "" {
		t.Errorf("unexpected package information data (-want +got):\n%s", diff)
	}
}

func TestReadResultChunkV0(t *testing.T) {
	data, exists, err := testStore(t, v0TestFile).ReadResultChunk(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error reading result chunk: %s", err)
	}
	if !exists {
		t.Errorf("expected result chunk to exist")
	}

	if path := data.DocumentPaths[types.ID("302")]; path != "protocol/protocol.go" {
		t.Errorf("unexpected document path. want=%s have=%s", "protocol/protocol.go", path)
	}

	expectedDocumentRanges := []types.DocumentIDRangeID{
		{DocumentID: "3981", RangeID: "4940"},
		{DocumentID: "3981", RangeID: "10759"},
		{DocumentID: "3981", RangeID: "10986"},
	}
	if diff := cmp.Diff(expectedDocumentRanges, data.DocumentIDRangeIDs[types.ID("14233")]); diff != "" {
		t.Errorf("unexpected document ranges (-want +got):\n%s", diff)
	}
}

func TestReadDefinitionsV0(t *testing.T) {
	definitions, totalCount, err := testStore(t, v0TestFile).ReadDefinitions(context.Background(), "gomod", "github.com/sourcegraph/lsif-go/protocol:Vertex", 3, 4)
	if err != nil {
		t.Fatalf("unexpected error getting definitions: %s", err)
	}
	if totalCount != 11 {
		t.Errorf("unexpected total count. want=%d have=%d", 11, totalCount)
	}

	expectedDefinitions := []types.Location{
		{URI: "protocol/protocol.go", StartLine: 334, StartCharacter: 1, EndLine: 334, EndCharacter: 7},
		{URI: "protocol/protocol.go", StartLine: 139, StartCharacter: 1, EndLine: 139, EndCharacter: 7},
		{URI: "protocol/protocol.go", StartLine: 384, StartCharacter: 1, EndLine: 384, EndCharacter: 7},
		{URI: "protocol/protocol.go", StartLine: 357, StartCharacter: 1, EndLine: 357, EndCharacter: 7},
	}
	if diff := cmp.Diff(expectedDefinitions, definitions); diff != "" {
		t.Errorf("unexpected definitions (-want +got):\n%s", diff)
	}
}

func TestReadReferencesV0(t *testing.T) {
	references, totalCount, err := testStore(t, v0TestFile).ReadReferences(context.Background(), "gomod", "golang.org/x/tools/go/packages:Package", 3, 4)
	if err != nil {
		t.Fatalf("unexpected error getting references: %s", err)
	}
	if totalCount != 25 {
		t.Errorf("unexpected total count. want=%d have=%d", 25, totalCount)
	}

	expectedReferences := []types.Location{
		{URI: "internal/index/helper.go", StartLine: 184, StartCharacter: 56, EndLine: 184, EndCharacter: 63},
		{URI: "internal/index/helper.go", StartLine: 35, StartCharacter: 56, EndLine: 35, EndCharacter: 63},
		{URI: "internal/index/helper.go", StartLine: 184, StartCharacter: 35, EndLine: 184, EndCharacter: 42},
		{URI: "internal/index/helper.go", StartLine: 48, StartCharacter: 44, EndLine: 48, EndCharacter: 51},
	}
	if diff := cmp.Diff(expectedReferences, references); diff != "" {
		t.Errorf("unexpected references (-want +got):\n%s", diff)
	}
}
