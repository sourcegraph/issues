package postgres

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

func init() {
	dbtesting.DBNameSuffix = "lsif-bundles"
}

func TestReadMeta(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)

	meta, err := testStore(t, 42, testFile).ReadMeta(context.Background())
	if err != nil {
		t.Fatalf("unexpected error reading meta: %s", err)
	}
	if meta.NumResultChunks != 4 {
		t.Errorf("unexpected numResultChunks. want=%d have=%d", 4, meta.NumResultChunks)
	}
}

func TestPathsWithPrefix(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)

	paths, err := testStore(t, 42, testFile).PathsWithPrefix(context.Background(), "internal/")
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

func TestReadDocument(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)

	data, exists, err := testStore(t, 42, testFile).ReadDocument(context.Background(), "protocol/writer.go")
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

func TestReadResultChunk(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)

	data, exists, err := testStore(t, 42, testFile).ReadResultChunk(context.Background(), 1)
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

func TestReadDefinitions(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)

	definitions, totalCount, err := testStore(t, 42, testFile).ReadDefinitions(context.Background(), "gomod", "github.com/sourcegraph/lsif-go/protocol:Vertex", 0, 0)
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

func TestReadReferences(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)

	references, totalCount, err := testStore(t, 42, testFile).ReadReferences(context.Background(), "gomod", "golang.org/x/tools/go/packages:Package", 3, 4)
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

func TestWrite(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)

	ctx := context.Background()
	store := testStore(t, 42, "")

	if err := store.CreateTables(ctx); err != nil {
		t.Fatalf("unexpected error while creating tables: %s", err)
	}

	if err := store.WriteMeta(ctx, types.MetaData{NumResultChunks: 7}); err != nil {
		t.Fatalf("unexpected error while writing: %s", err)
	}

	expectedDocumentData := types.DocumentData{
		Ranges: map[types.ID]types.RangeData{
			"r01": {StartLine: 1, StartCharacter: 2, EndLine: 3, EndCharacter: 4, DefinitionResultID: "x01", MonikerIDs: []types.ID{"m01", "m02"}},
			"r02": {StartLine: 2, StartCharacter: 3, EndLine: 4, EndCharacter: 5, ReferenceResultID: "x06", MonikerIDs: []types.ID{"m03", "m04"}},
			"r03": {StartLine: 3, StartCharacter: 4, EndLine: 5, EndCharacter: 6, DefinitionResultID: "x02"},
		},
		HoverResults: map[types.ID]string{},
		Monikers: map[types.ID]types.MonikerData{
			"m01": {Kind: "import", Scheme: "scheme A", Identifier: "ident A", PackageInformationID: "p01"},
			"m02": {Kind: "import", Scheme: "scheme B", Identifier: "ident B"},
			"m03": {Kind: "export", Scheme: "scheme C", Identifier: "ident C", PackageInformationID: "p02"},
			"m04": {Kind: "export", Scheme: "scheme D", Identifier: "ident D"},
		},
		PackageInformation: map[types.ID]types.PackageInformationData{
			"p01": {Name: "pkg A", Version: "0.1.0"},
			"p02": {Name: "pkg B", Version: "1.2.3"},
		},
	}

	documentCh := make(chan persistence.KeyedDocumentData, 1)
	documentCh <- persistence.KeyedDocumentData{
		Path:     "foo.go",
		Document: expectedDocumentData,
	}
	close(documentCh)

	if err := store.WriteDocuments(ctx, documentCh); err != nil {
		t.Fatalf("unexpected error while writing documents: %s", err)
	}

	expectedResultChunkData := types.ResultChunkData{
		DocumentPaths: map[types.ID]string{
			"d01": "foo.go",
			"d02": "bar.go",
			"d03": "baz.go",
		},
		DocumentIDRangeIDs: map[types.ID][]types.DocumentIDRangeID{
			"x01": {
				{DocumentID: "d01", RangeID: "r03"},
				{DocumentID: "d02", RangeID: "r04"},
				{DocumentID: "d03", RangeID: "r07"},
			},
			"x02": {
				{DocumentID: "d01", RangeID: "r02"},
				{DocumentID: "d02", RangeID: "r05"},
				{DocumentID: "d03", RangeID: "r08"},
			},
			"x03": {
				{DocumentID: "d01", RangeID: "r01"},
				{DocumentID: "d02", RangeID: "r06"},
				{DocumentID: "d03", RangeID: "r09"},
			},
		},
	}

	resultChunkCh := make(chan persistence.IndexedResultChunkData, 1)
	resultChunkCh <- persistence.IndexedResultChunkData{
		Index:       7,
		ResultChunk: expectedResultChunkData,
	}
	close(resultChunkCh)

	if err := store.WriteResultChunks(ctx, resultChunkCh); err != nil {
		t.Fatalf("unexpected error while writing result chunks: %s", err)
	}

	expectedDefinitions := []types.Location{
		{URI: "bar.go", StartLine: 4, StartCharacter: 5, EndLine: 6, EndCharacter: 7},
		{URI: "baz.go", StartLine: 7, StartCharacter: 8, EndLine: 9, EndCharacter: 0},
		{URI: "foo.go", StartLine: 3, StartCharacter: 4, EndLine: 5, EndCharacter: 6},
	}

	definitionsCh := make(chan types.MonikerLocations, 1)
	definitionsCh <- types.MonikerLocations{
		Scheme:     "scheme A",
		Identifier: "ident A",
		Locations:  expectedDefinitions,
	}
	close(definitionsCh)

	if err := store.WriteDefinitions(ctx, definitionsCh); err != nil {
		t.Fatalf("unexpected error while writing definitions: %s", err)
	}

	expectedReferences := []types.Location{
		{URI: "baz.go", StartLine: 7, StartCharacter: 8, EndLine: 9, EndCharacter: 0},
		{URI: "baz.go", StartLine: 9, StartCharacter: 0, EndLine: 1, EndCharacter: 2},
		{URI: "foo.go", StartLine: 3, StartCharacter: 4, EndLine: 5, EndCharacter: 6},
	}

	referencesCh := make(chan types.MonikerLocations, 1)
	referencesCh <- types.MonikerLocations{
		Scheme:     "scheme C",
		Identifier: "ident C",
		Locations:  expectedReferences,
	}
	close(referencesCh)

	if err := store.WriteReferences(ctx, referencesCh); err != nil {
		t.Fatalf("unexpected error while writing references: %s", err)
	}

	if err := store.Done(nil); err != nil {
		t.Fatalf("unexpected error closing transaction: %s", err)
	}

	meta, err := store.ReadMeta(ctx)
	if err != nil {
		t.Fatalf("unexpected error reading from database: %s", err)
	}
	if meta.NumResultChunks != 7 {
		t.Errorf("unexpected num result chunks. want=%d have=%d", 7, meta.NumResultChunks)
	}

	documentData, _, err := store.ReadDocument(ctx, "foo.go")
	if err != nil {
		t.Fatalf("unexpected error reading from database: %s", err)
	}
	if diff := cmp.Diff(expectedDocumentData, documentData); diff != "" {
		t.Errorf("unexpected document data (-want +got):\n%s", diff)
	}

	resultChunkData, _, err := store.ReadResultChunk(ctx, 7)
	if err != nil {
		t.Fatalf("unexpected error reading from database: %s", err)
	}
	if diff := cmp.Diff(expectedResultChunkData, resultChunkData); diff != "" {
		t.Errorf("unexpected result chunk data (-want +got):\n%s", diff)
	}

	definitions, _, err := store.ReadDefinitions(ctx, "scheme A", "ident A", 0, 100)
	if err != nil {
		t.Fatalf("unexpected error reading from database: %s", err)
	}
	if diff := cmp.Diff(expectedDefinitions, definitions); diff != "" {
		t.Errorf("unexpected definitions (-want +got):\n%s", diff)
	}

	references, _, err := store.ReadReferences(ctx, "scheme C", "ident C", 0, 100)
	if err != nil {
		t.Fatalf("unexpected error reading from database: %s", err)
	}
	if diff := cmp.Diff(expectedReferences, references); diff != "" {
		t.Errorf("unexpected references (-want +got):\n%s", diff)
	}
}

const testFile = "../sqlite/testdata/lsif-go@5bc35c78.lsif.db"

func testStore(t *testing.T, id int, filename string) persistence.Store {
	store := NewStore(dbconn.Global, id)
	t.Cleanup(func() { _ = store.Close(nil) })

	if filename != "" {
		if err := MigrateBundleToPostgres(context.Background(), id, filename, dbconn.Global); err != nil {
			t.Fatalf("unexpected error migrating test bundle: %s", err)
		}
	}

	// Wrap in observed, as that's how it's used in production
	return persistence.NewObserved(store, &observation.TestContext)
}
