package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/database"
	databasemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/database/mocks"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence"
	persistencemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/mocks"
	postgresreader "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/postgres"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
)

func TestExists(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/dbs/42/exists", map[string]string{
			"path": "main.go",
		})

		_, _ = w.Write([]byte(`true`))
	}))
	defer ts.Close()

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	exists, err := client.Exists(context.Background(), "main.go")
	if err != nil {
		t.Fatalf("unexpected error querying exists: %s", err)
	} else if !exists {
		t.Errorf("unexpected path to exist")
	}
}

func TestExistsNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	_, err := client.Exists(context.Background(), "main.go")
	if err != ErrNotFound {
		t.Fatalf("unexpected error. want=%q have=%q", ErrNotFound, err)
	}
}

func TestExistsBadResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	_, err := client.Exists(context.Background(), "main.go")
	if err == nil {
		t.Fatalf("unexpected nil error querying exists")
	}
}

func TestExistsDB(t *testing.T) {
	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, nil)
	mockDatabase := databasemocks.NewMockDatabase()
	mockDatabase.ExistsFunc.SetDefaultReturn(true, nil)
	databaseOpener := func(ctx context.Context, filename string, s persistence.Store) (database.Database, error) {
		return mockDatabase, nil
	}
	base := &bundleManagerClientImpl{}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore, databaseOpener: databaseOpener}
	exists, err := client.Exists(context.Background(), "main.go")
	if err != nil {
		t.Fatalf("unexpected error querying exists: %s", err)
	} else if !exists {
		t.Errorf("unexpected path to exist")
	}
}

func TestRanges(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/dbs/42/ranges", map[string]string{
			"path":      "main.go",
			"startLine": "15",
			"endLine":   "20",
		})

		_, _ = w.Write([]byte(`[
			{
				"range": {"start": {"line": 1, "character": 2}, "end": {"line": 3, "character": 4}},
				"definitions": [],
				"references": [],
				"hoverText": ""
			},
			{
				"range": {"start": {"line": 2, "character": 3}, "end": {"line": 4, "character": 5}},
				"definitions": [{"path": "foo.go", "range": {"start": {"line": 10, "character": 20}, "end": {"line": 30, "character": 40}}}],
				"references": [{"path": "bar.go", "range": {"start": {"line": 100, "character": 200}, "end": {"line": 300, "character": 400}}}],
				"hoverText": "ht2"
			},
			{
				"range": {"start": {"line": 3, "character": 4}, "end": {"line": 5, "character": 6}},
				"definitions": [{"path": "bar.go", "range": {"start": {"line": 11, "character": 21}, "end": {"line": 31, "character": 41}}}],
				"references": [{"path": "foo.go", "range": {"start": {"line": 101, "character": 201}, "end": {"line": 301, "character": 401}}}],
				"hoverText": "ht3"
			}
		]`))
	}))
	defer ts.Close()

	expected := []CodeIntelligenceRange{
		{
			Range:       Range{Start: Position{1, 2}, End: Position{3, 4}},
			Definitions: []Location{},
			References:  []Location{},
			HoverText:   "",
		},
		{
			Range:       Range{Start: Position{2, 3}, End: Position{4, 5}},
			Definitions: []Location{{Path: "foo.go", Range: Range{Start: Position{10, 20}, End: Position{30, 40}}}},
			References:  []Location{{Path: "bar.go", Range: Range{Start: Position{100, 200}, End: Position{300, 400}}}},
			HoverText:   "ht2",
		},
		{
			Range:       Range{Start: Position{3, 4}, End: Position{5, 6}},
			Definitions: []Location{{Path: "bar.go", Range: Range{Start: Position{11, 21}, End: Position{31, 41}}}},
			References:  []Location{{Path: "foo.go", Range: Range{Start: Position{101, 201}, End: Position{301, 401}}}},
			HoverText:   "ht3",
		},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	ranges, err := client.Ranges(context.Background(), "main.go", 15, 20)
	if err != nil {
		t.Fatalf("unexpected error querying ranges: %s", err)
	} else if diff := cmp.Diff(expected, ranges); diff != "" {
		t.Errorf("unexpected ranges (-want +got):\n%s", diff)
	}
}

func TestRangesNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	_, err := client.Ranges(context.Background(), "main.go", 15, 20)
	if err != ErrNotFound {
		t.Fatalf("unexpected error. want=%q have=%q", ErrNotFound, err)
	}
}

func TestRangesBadResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	_, err := client.Ranges(context.Background(), "main.go", 15, 20)
	if err == nil {
		t.Fatalf("unexpected nil error querying ranges")
	}
}

func TestRangesDB(t *testing.T) {
	expected := []CodeIntelligenceRange{
		{
			Range:       Range{Start: Position{1, 2}, End: Position{3, 4}},
			Definitions: []Location{},
			References:  []Location{},
			HoverText:   "",
		},
		{
			Range:       Range{Start: Position{2, 3}, End: Position{4, 5}},
			Definitions: []Location{{Path: "foo.go", Range: Range{Start: Position{10, 20}, End: Position{30, 40}}}},
			References:  []Location{{Path: "bar.go", Range: Range{Start: Position{100, 200}, End: Position{300, 400}}}},
			HoverText:   "ht2",
		},
		{
			Range:       Range{Start: Position{3, 4}, End: Position{5, 6}},
			Definitions: []Location{{Path: "bar.go", Range: Range{Start: Position{11, 21}, End: Position{31, 41}}}},
			References:  []Location{{Path: "foo.go", Range: Range{Start: Position{101, 201}, End: Position{301, 401}}}},
			HoverText:   "ht3",
		},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, nil)
	mockDatabase := databasemocks.NewMockDatabase()
	mockDatabase.RangesFunc.SetDefaultReturn(expected, nil)
	databaseOpener := func(ctx context.Context, filename string, s persistence.Store) (database.Database, error) {
		return mockDatabase, nil
	}
	base := &bundleManagerClientImpl{}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore, databaseOpener: databaseOpener}
	ranges, err := client.Ranges(context.Background(), "main.go", 15, 20)
	if err != nil {
		t.Fatalf("unexpected error querying ranges: %s", err)
	} else if diff := cmp.Diff(expected, ranges); diff != "" {
		t.Errorf("unexpected ranges (-want +got):\n%s", diff)
	}
}

func TestDefinitions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/dbs/42/definitions", map[string]string{
			"path":      "main.go",
			"line":      "10",
			"character": "20",
		})

		_, _ = w.Write([]byte(`[
			{"path": "foo.go", "range": {"start": {"line": 1, "character": 2}, "end": {"line": 3, "character": 4}}},
			{"path": "bar.go", "range": {"start": {"line": 5, "character": 6}, "end": {"line": 7, "character": 8}}}
		]`))
	}))
	defer ts.Close()

	expected := []Location{
		{DumpID: 42, Path: "foo.go", Range: Range{Start: Position{1, 2}, End: Position{3, 4}}},
		{DumpID: 42, Path: "bar.go", Range: Range{Start: Position{5, 6}, End: Position{7, 8}}},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	definitions, err := client.Definitions(context.Background(), "main.go", 10, 20)
	if err != nil {
		t.Fatalf("unexpected error querying definitions: %s", err)
	} else if diff := cmp.Diff(expected, definitions); diff != "" {
		t.Errorf("unexpected definitions (-want +got):\n%s", diff)
	}
}

func TestDefinitionsDB(t *testing.T) {
	expected := []Location{
		{DumpID: 42, Path: "foo.go", Range: Range{Start: Position{1, 2}, End: Position{3, 4}}},
		{DumpID: 42, Path: "bar.go", Range: Range{Start: Position{5, 6}, End: Position{7, 8}}},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, nil)
	mockDatabase := databasemocks.NewMockDatabase()
	mockDatabase.DefinitionsFunc.SetDefaultReturn(expected, nil)
	databaseOpener := func(ctx context.Context, filename string, s persistence.Store) (database.Database, error) {
		return mockDatabase, nil
	}
	base := &bundleManagerClientImpl{}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore, databaseOpener: databaseOpener}
	definitions, err := client.Definitions(context.Background(), "main.go", 10, 20)
	if err != nil {
		t.Fatalf("unexpected error querying definitions: %s", err)
	} else if diff := cmp.Diff(expected, definitions); diff != "" {
		t.Errorf("unexpected definitions (-want +got):\n%s", diff)
	}
}

func TestReferences(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/dbs/42/references", map[string]string{
			"path":      "main.go",
			"line":      "10",
			"character": "20",
		})

		_, _ = w.Write([]byte(`[
			{"path": "foo.go", "range": {"start": {"line": 1, "character": 2}, "end": {"line": 3, "character": 4}}},
			{"path": "bar.go", "range": {"start": {"line": 5, "character": 6}, "end": {"line": 7, "character": 8}}}
		]`))
	}))
	defer ts.Close()

	expected := []Location{
		{DumpID: 42, Path: "foo.go", Range: Range{Start: Position{1, 2}, End: Position{3, 4}}},
		{DumpID: 42, Path: "bar.go", Range: Range{Start: Position{5, 6}, End: Position{7, 8}}},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	references, err := client.References(context.Background(), "main.go", 10, 20)
	if err != nil {
		t.Fatalf("unexpected error querying references: %s", err)
	} else if diff := cmp.Diff(expected, references); diff != "" {
		t.Errorf("unexpected references (-want +got):\n%s", diff)
	}
}

func TestReferencesDB(t *testing.T) {
	expected := []Location{
		{DumpID: 42, Path: "foo.go", Range: Range{Start: Position{1, 2}, End: Position{3, 4}}},
		{DumpID: 42, Path: "bar.go", Range: Range{Start: Position{5, 6}, End: Position{7, 8}}},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, nil)
	mockDatabase := databasemocks.NewMockDatabase()
	mockDatabase.ReferencesFunc.SetDefaultReturn(expected, nil)
	databaseOpener := func(ctx context.Context, filename string, s persistence.Store) (database.Database, error) {
		return mockDatabase, nil
	}
	base := &bundleManagerClientImpl{}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore, databaseOpener: databaseOpener}
	references, err := client.References(context.Background(), "main.go", 10, 20)
	if err != nil {
		t.Fatalf("unexpected error querying references: %s", err)
	} else if diff := cmp.Diff(expected, references); diff != "" {
		t.Errorf("unexpected references (-want +got):\n%s", diff)
	}
}

func TestHover(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/dbs/42/hover", map[string]string{
			"path":      "main.go",
			"line":      "10",
			"character": "20",
		})

		_, _ = w.Write([]byte(`{
			"text": "starts the program",
			"range": {"start": {"line": 1, "character": 2}, "end": {"line": 3, "character": 4}}
		}`))
	}))
	defer ts.Close()

	expectedText := "starts the program"
	expectedRange := Range{
		Start: Position{1, 2},
		End:   Position{3, 4},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	text, r, exists, err := client.Hover(context.Background(), "main.go", 10, 20)
	if err != nil {
		t.Fatalf("unexpected error querying hover: %s", err)
	}

	if !exists {
		t.Errorf("expected hover text to exist")
	} else {
		if text != expectedText {
			t.Errorf("unexpected hover text. want=%v have=%v", expectedText, text)
		} else if diff := cmp.Diff(expectedRange, r); diff != "" {
			t.Errorf("unexpected hover range (-want +got):\n%s", diff)
		}
	}
}

func TestHoverNull(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/dbs/42/hover", map[string]string{
			"path":      "main.go",
			"line":      "10",
			"character": "20",
		})

		_, _ = w.Write([]byte(`null`))
	}))
	defer ts.Close()

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	_, _, exists, err := client.Hover(context.Background(), "main.go", 10, 20)
	if err != nil {
		t.Fatalf("unexpected error querying hover: %s", err)
	} else if exists {
		t.Errorf("unexpected hover text")
	}
}

func TestHoverDB(t *testing.T) {
	expectedText := "starts the program"
	expectedRange := Range{
		Start: Position{1, 2},
		End:   Position{3, 4},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, nil)
	mockDatabase := databasemocks.NewMockDatabase()
	mockDatabase.HoverFunc.SetDefaultReturn(expectedText, expectedRange, true, nil)
	databaseOpener := func(ctx context.Context, filename string, s persistence.Store) (database.Database, error) {
		return mockDatabase, nil
	}
	base := &bundleManagerClientImpl{}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore, databaseOpener: databaseOpener}
	text, r, exists, err := client.Hover(context.Background(), "main.go", 10, 20)
	if err != nil {
		t.Fatalf("unexpected error querying hover: %s", err)
	}

	if !exists {
		t.Errorf("expected hover text to exist")
	} else {
		if text != expectedText {
			t.Errorf("unexpected hover text. want=%v have=%v", expectedText, text)
		} else if diff := cmp.Diff(expectedRange, r); diff != "" {
			t.Errorf("unexpected hover range (-want +got):\n%s", diff)
		}
	}
}

func TestDiagnostics(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/dbs/42/diagnostics", map[string]string{
			"prefix": "internal/",
			"skip":   "1",
			"take":   "3",
		})

		_, _ = w.Write([]byte(`{
			"count": 5,
			"diagnostics": [
				{"path": "internal/foo.go", "severity": 1, "code": "c1", "message": "m1", "source": "s1", "startLine": 11, "startCharacter": 12, "endLine": 13, "endCharacter": 14},
				{"path": "internal/bar.go", "severity": 2, "code": "c2", "message": "m2", "source": "s2", "startLine": 21, "startCharacter": 22, "endLine": 23, "endCharacter": 24},
				{"path": "internal/baz.go", "severity": 3, "code": "c3", "message": "m3", "source": "s3", "startLine": 31, "startCharacter": 32, "endLine": 33, "endCharacter": 34}
			]
		}`))
	}))
	defer ts.Close()

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	diagnostics, totalCount, err := client.Diagnostics(context.Background(), "internal/", 1, 3)
	if err != nil {
		t.Fatalf("unexpected error querying diagnostics: %s", err)
	}

	expectedDiagnostics := []Diagnostic{
		{
			DumpID:         42,
			Path:           "internal/foo.go",
			Severity:       1,
			Code:           "c1",
			Message:        "m1",
			Source:         "s1",
			StartLine:      11,
			StartCharacter: 12,
			EndLine:        13,
			EndCharacter:   14,
		},
		{
			DumpID:         42,
			Path:           "internal/bar.go",
			Severity:       2,
			Code:           "c2",
			Message:        "m2",
			Source:         "s2",
			StartLine:      21,
			StartCharacter: 22,
			EndLine:        23,
			EndCharacter:   24,
		},
		{
			DumpID:         42,
			Path:           "internal/baz.go",
			Severity:       3,
			Code:           "c3",
			Message:        "m3",
			Source:         "s3",
			StartLine:      31,
			StartCharacter: 32,
			EndLine:        33,
			EndCharacter:   34,
		},
	}
	if diff := cmp.Diff(expectedDiagnostics, diagnostics); diff != "" {
		t.Errorf("unexpected moniker data (-want +got):\n%s", diff)
	}

	if totalCount != 5 {
		t.Errorf("unexpected total count. want=%d have=%d", 5, totalCount)
	}
}

func TestDiagnosticsDB(t *testing.T) {
	expected := []Diagnostic{
		{
			DumpID:         42,
			Path:           "internal/foo.go",
			Severity:       1,
			Code:           "c1",
			Message:        "m1",
			Source:         "s1",
			StartLine:      11,
			StartCharacter: 12,
			EndLine:        13,
			EndCharacter:   14,
		},
		{
			DumpID:         42,
			Path:           "internal/bar.go",
			Severity:       2,
			Code:           "c2",
			Message:        "m2",
			Source:         "s2",
			StartLine:      21,
			StartCharacter: 22,
			EndLine:        23,
			EndCharacter:   24,
		},
		{
			DumpID:         42,
			Path:           "internal/baz.go",
			Severity:       3,
			Code:           "c3",
			Message:        "m3",
			Source:         "s3",
			StartLine:      31,
			StartCharacter: 32,
			EndLine:        33,
			EndCharacter:   34,
		},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, nil)
	mockDatabase := databasemocks.NewMockDatabase()
	mockDatabase.DiagnosticsFunc.SetDefaultReturn(expected, 5, nil)
	databaseOpener := func(ctx context.Context, filename string, s persistence.Store) (database.Database, error) {
		return mockDatabase, nil
	}
	base := &bundleManagerClientImpl{}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore, databaseOpener: databaseOpener}
	diagnostics, totalCount, err := client.Diagnostics(context.Background(), "internal/", 1, 3)
	if err != nil {
		t.Fatalf("unexpected error querying diagnostics: %s", err)
	}

	if diff := cmp.Diff(expected, diagnostics); diff != "" {
		t.Errorf("unexpected moniker data (-want +got):\n%s", diff)
	}

	if totalCount != 5 {
		t.Errorf("unexpected total count. want=%d have=%d", 5, totalCount)
	}
}

func TestMonikersByPosition(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/dbs/42/monikersByPosition", map[string]string{
			"path":      "main.go",
			"line":      "10",
			"character": "20",
		})

		_, _ = w.Write([]byte(`[
			[{
				"kind": "import",
				"scheme": "gomod",
				"identifier": "pad1"
			}],
			[{
				"kind": "import",
				"scheme": "gomod",
				"identifier": "pad2",
				"packageInformationID": "123"
			}, {
				"kind": "export",
				"scheme": "gomod",
				"identifier": "pad2",
				"packageInformationID": "123"
			}]
		]`))
	}))
	defer ts.Close()

	expected := [][]MonikerData{
		{
			{Kind: "import", Scheme: "gomod", Identifier: "pad1"},
		},
		{
			{Kind: "import", Scheme: "gomod", Identifier: "pad2", PackageInformationID: "123"},
			{Kind: "export", Scheme: "gomod", Identifier: "pad2", PackageInformationID: "123"},
		},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	monikers, err := client.MonikersByPosition(context.Background(), "main.go", 10, 20)
	if err != nil {
		t.Fatalf("unexpected error querying monikers by position: %s", err)
	} else if diff := cmp.Diff(expected, monikers); diff != "" {
		t.Errorf("unexpected moniker data (-want +got):\n%s", diff)
	}
}

func TestMonikersByPositionDB(t *testing.T) {
	expected := [][]MonikerData{
		{
			{Kind: "import", Scheme: "gomod", Identifier: "pad1"},
		},
		{
			{Kind: "import", Scheme: "gomod", Identifier: "pad2", PackageInformationID: "123"},
			{Kind: "export", Scheme: "gomod", Identifier: "pad2", PackageInformationID: "123"},
		},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, nil)
	mockDatabase := databasemocks.NewMockDatabase()
	mockDatabase.MonikersByPositionFunc.SetDefaultReturn(expected, nil)
	databaseOpener := func(ctx context.Context, filename string, s persistence.Store) (database.Database, error) {
		return mockDatabase, nil
	}
	base := &bundleManagerClientImpl{}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore, databaseOpener: databaseOpener}
	monikers, err := client.MonikersByPosition(context.Background(), "main.go", 10, 20)
	if err != nil {
		t.Fatalf("unexpected error querying monikers by position: %s", err)
	} else if diff := cmp.Diff(expected, monikers); diff != "" {
		t.Errorf("unexpected moniker data (-want +got):\n%s", diff)
	}
}

func TestMonikerResults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/dbs/42/monikerResults", map[string]string{
			"modelType":  "definition",
			"scheme":     "gomod",
			"identifier": "leftpad",
			"take":       "25",
		})

		_, _ = w.Write([]byte(`{
			"locations": [
				{"path": "foo.go", "range": {"start": {"line": 1, "character": 2}, "end": {"line": 3, "character": 4}}},
				{"path": "bar.go", "range": {"start": {"line": 5, "character": 6}, "end": {"line": 7, "character": 8}}}
			],
			"count": 5
		}`))

	}))
	defer ts.Close()

	expected := []Location{
		{DumpID: 42, Path: "foo.go", Range: Range{Start: Position{1, 2}, End: Position{3, 4}}},
		{DumpID: 42, Path: "bar.go", Range: Range{Start: Position{5, 6}, End: Position{7, 8}}},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	locations, count, err := client.MonikerResults(context.Background(), "definition", "gomod", "leftpad", 0, 25)
	if err != nil {
		t.Fatalf("unexpected error querying moniker results: %s", err)
	}
	if count != 5 {
		t.Errorf("unexpected count. want=%v have=%v", 5, count)
	}
	if diff := cmp.Diff(expected, locations); diff != "" {
		t.Errorf("unexpected locations (-want +got):\n%s", diff)
	}
}

func TestMonikerResultsDB(t *testing.T) {
	expected := []Location{
		{DumpID: 42, Path: "foo.go", Range: Range{Start: Position{1, 2}, End: Position{3, 4}}},
		{DumpID: 42, Path: "bar.go", Range: Range{Start: Position{5, 6}, End: Position{7, 8}}},
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, nil)
	mockDatabase := databasemocks.NewMockDatabase()
	mockDatabase.MonikerResultsFunc.SetDefaultReturn(expected, 5, nil)
	databaseOpener := func(ctx context.Context, filename string, s persistence.Store) (database.Database, error) {
		return mockDatabase, nil
	}
	base := &bundleManagerClientImpl{}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore, databaseOpener: databaseOpener}
	locations, count, err := client.MonikerResults(context.Background(), "definition", "gomod", "leftpad", 0, 25)
	if err != nil {
		t.Fatalf("unexpected error querying moniker results: %s", err)
	}
	if count != 5 {
		t.Errorf("unexpected count. want=%v have=%v", 5, count)
	}
	if diff := cmp.Diff(expected, locations); diff != "" {
		t.Errorf("unexpected locations (-want +got):\n%s", diff)
	}
}

func TestPackageInformation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/dbs/42/packageInformation", map[string]string{
			"path":                 "main.go",
			"packageInformationId": "123",
		})

		_, _ = w.Write([]byte(`{"name": "leftpad", "version": "0.1.0"}`))
	}))
	defer ts.Close()

	expected := PackageInformationData{
		Name:    "leftpad",
		Version: "0.1.0",
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, postgresreader.ErrNoMetadata)
	base := &bundleManagerClientImpl{bundleManagerURL: ts.URL}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore}
	packageInformation, err := client.PackageInformation(context.Background(), "main.go", "123")
	if err != nil {
		t.Fatalf("unexpected error querying package information: %s", err)
	} else if diff := cmp.Diff(expected, packageInformation); diff != "" {
		t.Errorf("unexpected package information (-want +got):\n%s", diff)
	}
}

func TestPackageInformationDB(t *testing.T) {
	expected := PackageInformationData{
		Name:    "leftpad",
		Version: "0.1.0",
	}

	mockStore := persistencemocks.NewMockStore()
	mockStore.ReadMetaFunc.SetDefaultReturn(types.MetaData{}, nil)
	mockDatabase := databasemocks.NewMockDatabase()
	mockDatabase.PackageInformationFunc.SetDefaultReturn(expected, true, nil)
	databaseOpener := func(ctx context.Context, filename string, s persistence.Store) (database.Database, error) {
		return mockDatabase, nil
	}
	base := &bundleManagerClientImpl{}
	client := &bundleClientImpl{base: base, bundleID: 42, store: mockStore, databaseOpener: databaseOpener}
	packageInformation, err := client.PackageInformation(context.Background(), "main.go", "123")
	if err != nil {
		t.Fatalf("unexpected error querying package information: %s", err)
	} else if diff := cmp.Diff(expected, packageInformation); diff != "" {
		t.Errorf("unexpected package information (-want +got):\n%s", diff)
	}
}

func assertRequest(t *testing.T, r *http.Request, expectedMethod, expectedPath string, expectedQuery map[string]string) {
	if r.Method != expectedMethod {
		t.Errorf("unexpected method. want=%s have=%s", expectedMethod, r.Method)
	}
	if r.URL.Path != expectedPath {
		t.Errorf("unexpected path. want=%s have=%s", expectedPath, r.URL.Path)
	}
	if !compareQuery(r.URL.Query(), expectedQuery) {
		t.Errorf("unexpected query. want=%v have=%s", expectedQuery, r.URL.Query().Encode())
	}
}

func compareQuery(query url.Values, expected map[string]string) bool {
	values := map[string]string{}
	for k, v := range query {
		values[k] = v[0]
	}

	return cmp.Diff(expected, values) == ""
}
