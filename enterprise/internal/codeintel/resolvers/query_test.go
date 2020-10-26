package resolvers

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	codeintelapi "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/api"
	apimocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/api/mocks"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	bundlemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client/mocks"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	storemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store/mocks"
)

func TestRanges(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockBundleManagerClient := bundlemocks.NewMockBundleManagerClient()
	mockCodeIntelAPI := apimocks.NewMockCodeIntelAPI()
	mockPositionAdjuster := NewMockPositionAdjuster()

	// path can be translated for subsequent dumps
	mockPositionAdjuster.AdjustPathFunc.SetDefaultReturn("renamed.go", true, nil)

	// first requested dump (dump 42) has no equivalent path
	mockPositionAdjuster.AdjustPathFunc.PushReturn("", false, nil)

	// second requested dump (dump 44) has some data
	mockCodeIntelAPI.RangesFunc.PushReturn([]codeintelapi.ResolvedCodeIntelligenceRange{
		{
			Range:       bundles.Range{Start: bundles.Position{Line: 11, Character: 12}, End: bundles.Position{Line: 13, Character: 14}},
			Definitions: []codeintelapi.ResolvedLocation{{Dump: store.Dump{ID: 44, RepositoryID: 50}, Path: "p1.go", Range: bundles.Range{Start: bundles.Position{Line: 111, Character: 121}, End: bundles.Position{Line: 131, Character: 141}}}},
			References:  []codeintelapi.ResolvedLocation{{Dump: store.Dump{ID: 44, RepositoryID: 50}, Path: "p1.go", Range: bundles.Range{Start: bundles.Position{Line: 112, Character: 122}, End: bundles.Position{Line: 132, Character: 142}}}},
			HoverText:   "ht1",
		},
		{
			Range:       bundles.Range{Start: bundles.Position{Line: 21, Character: 22}, End: bundles.Position{Line: 23, Character: 24}},
			Definitions: []codeintelapi.ResolvedLocation{{Dump: store.Dump{ID: 44, RepositoryID: 50}, Path: "p2.go", Range: bundles.Range{Start: bundles.Position{Line: 211, Character: 221}, End: bundles.Position{Line: 231, Character: 241}}}},
			References:  []codeintelapi.ResolvedLocation{{Dump: store.Dump{ID: 44, RepositoryID: 50}, Path: "p2.go", Range: bundles.Range{Start: bundles.Position{Line: 212, Character: 222}, End: bundles.Position{Line: 232, Character: 242}}}},
			HoverText:   "ht2",
		},
		{
			Range:       bundles.Range{Start: bundles.Position{Line: 31, Character: 32}, End: bundles.Position{Line: 33, Character: 34}},
			Definitions: []codeintelapi.ResolvedLocation{{Dump: store.Dump{ID: 44, RepositoryID: 50}, Path: "p3.go", Range: bundles.Range{Start: bundles.Position{Line: 311, Character: 321}, End: bundles.Position{Line: 331, Character: 341}}}},
			References:  []codeintelapi.ResolvedLocation{{Dump: store.Dump{ID: 44, RepositoryID: 50}, Path: "p3.go", Range: bundles.Range{Start: bundles.Position{Line: 312, Character: 322}, End: bundles.Position{Line: 332, Character: 342}}}},
			HoverText:   "ht3",
		},
	}, nil)

	// first requested dump (dump 43) has no data
	mockCodeIntelAPI.RangesFunc.PushReturn(nil, nil)

	mockPositionAdjuster.AdjustRangeFunc.SetDefaultHook(func(ctx context.Context, path, commit string, r bundles.Range, reverse bool) (string, bundles.Range, bool, error) {
		return path, bundles.Range{
			Start: bundles.Position{Line: r.Start.Line * 10, Character: r.Start.Character * 10},
			End:   bundles.Position{Line: r.End.Line * 10, Character: r.End.Character * 10},
		}, true, nil
	})

	queryResolver := NewQueryResolver(
		mockStore,
		mockBundleManagerClient,
		mockCodeIntelAPI,
		mockPositionAdjuster,
		50,
		"deadbeef2",
		"/foo/bar.go",
		[]store.Dump{
			{ID: 42, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 43, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 44, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 45, RepositoryID: 50, Commit: "deadbeef1"},
		},
	)

	ranges, err := queryResolver.Ranges(context.Background(), 10, 20)
	if err != nil {
		t.Fatalf("unexpected error resolving ranges: %s", err)
	}

	expectedRanges := []AdjustedCodeIntelligenceRange{
		{
			Range: bundles.Range{Start: bundles.Position{Line: 110, Character: 120}, End: bundles.Position{Line: 130, Character: 140}},
			Definitions: []AdjustedLocation{
				{
					Dump:           store.Dump{ID: 44, RepositoryID: 50},
					Path:           "p1.go",
					AdjustedCommit: "deadbeef2",
					AdjustedRange:  bundles.Range{Start: bundles.Position{Line: 1110, Character: 1210}, End: bundles.Position{Line: 1310, Character: 1410}},
				},
			},
			References: []AdjustedLocation{
				{
					Dump:           store.Dump{ID: 44, RepositoryID: 50},
					Path:           "p1.go",
					AdjustedCommit: "deadbeef2",
					AdjustedRange:  bundles.Range{Start: bundles.Position{Line: 1120, Character: 1220}, End: bundles.Position{Line: 1320, Character: 1420}},
				},
			},
			HoverText: "ht1",
		},
		{
			Range: bundles.Range{Start: bundles.Position{Line: 210, Character: 220}, End: bundles.Position{Line: 230, Character: 240}},
			Definitions: []AdjustedLocation{
				{
					Dump:           store.Dump{ID: 44, RepositoryID: 50},
					Path:           "p2.go",
					AdjustedCommit: "deadbeef2",
					AdjustedRange:  bundles.Range{Start: bundles.Position{Line: 2110, Character: 2210}, End: bundles.Position{Line: 2310, Character: 2410}},
				},
			},
			References: []AdjustedLocation{
				{
					Dump:           store.Dump{ID: 44, RepositoryID: 50},
					Path:           "p2.go",
					AdjustedCommit: "deadbeef2",
					AdjustedRange:  bundles.Range{Start: bundles.Position{Line: 2120, Character: 2220}, End: bundles.Position{Line: 2320, Character: 2420}},
				},
			},
			HoverText: "ht2",
		},
		{
			Range: bundles.Range{Start: bundles.Position{Line: 310, Character: 320}, End: bundles.Position{Line: 330, Character: 340}},
			Definitions: []AdjustedLocation{
				{
					Dump:           store.Dump{ID: 44, RepositoryID: 50},
					Path:           "p3.go",
					AdjustedCommit: "deadbeef2",
					AdjustedRange:  bundles.Range{Start: bundles.Position{Line: 3110, Character: 3210}, End: bundles.Position{Line: 3310, Character: 3410}},
				},
			},
			References: []AdjustedLocation{
				{
					Dump:           store.Dump{ID: 44, RepositoryID: 50},
					Path:           "p3.go",
					AdjustedCommit: "deadbeef2",
					AdjustedRange:  bundles.Range{Start: bundles.Position{Line: 3120, Character: 3220}, End: bundles.Position{Line: 3320, Character: 3420}},
				},
			},
			HoverText: "ht3",
		},
	}
	if diff := cmp.Diff(expectedRanges, ranges); diff != "" {
		t.Errorf("unexpected ranges (-want +got):\n%s", diff)
	}
}

func TestDefinitions(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockBundleManagerClient := bundlemocks.NewMockBundleManagerClient()
	mockCodeIntelAPI := apimocks.NewMockCodeIntelAPI()
	mockPositionAdjuster := NewMockPositionAdjuster()

	// position can be translated for subsequent dumps
	mockPositionAdjuster.AdjustPositionFunc.SetDefaultReturn("", bundles.Position{Line: 20, Character: 15}, true, nil)

	// first requested dump (dump 42) has no equivalent position
	mockPositionAdjuster.AdjustPositionFunc.PushReturn("", bundles.Position{}, false, nil)

	mockCodeIntelAPI.DefinitionsFunc.SetDefaultReturn([]codeintelapi.ResolvedLocation{
		{
			Dump: store.Dump{ID: 44, RepositoryID: 50},
			Path: "p1.go",
			Range: bundles.Range{
				Start: bundles.Position{Line: 11, Character: 12},
				End:   bundles.Position{Line: 13, Character: 14},
			},
		},
		{
			Dump: store.Dump{ID: 44, RepositoryID: 50},
			Path: "p2.go",
			Range: bundles.Range{
				Start: bundles.Position{Line: 21, Character: 22},
				End:   bundles.Position{Line: 23, Character: 24},
			},
		},
		{
			Dump: store.Dump{ID: 44, RepositoryID: 50},
			Path: "p3.go",
			Range: bundles.Range{
				Start: bundles.Position{Line: 31, Character: 32},
				End:   bundles.Position{Line: 33, Character: 34},
			},
		},
	}, nil)

	// first requested dump (dump 43) has no definitions
	mockCodeIntelAPI.DefinitionsFunc.PushReturn(nil, nil)

	mockPositionAdjuster.AdjustRangeFunc.SetDefaultHook(func(ctx context.Context, path, commit string, r bundles.Range, reverse bool) (string, bundles.Range, bool, error) {
		return path, bundles.Range{
			Start: bundles.Position{Line: r.Start.Line * 10, Character: r.Start.Character * 10},
			End:   bundles.Position{Line: r.End.Line * 10, Character: r.End.Character * 10},
		}, true, nil
	})

	queryResolver := NewQueryResolver(
		mockStore,
		mockBundleManagerClient,
		mockCodeIntelAPI,
		mockPositionAdjuster,
		50,
		"deadbeef2",
		"/foo/bar.go",
		[]store.Dump{
			{ID: 42, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 43, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 44, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 45, RepositoryID: 50, Commit: "deadbeef1"},
		},
	)

	definitions, err := queryResolver.Definitions(context.Background(), 10, 15)
	if err != nil {
		t.Fatalf("unexpected error resolving definitions: %s", err)
	}

	expectedDefinitions := []AdjustedLocation{
		{
			Dump:           store.Dump{ID: 44, RepositoryID: 50},
			Path:           "p1.go",
			AdjustedCommit: "deadbeef2",
			AdjustedRange: bundles.Range{
				Start: bundles.Position{Line: 110, Character: 120},
				End:   bundles.Position{Line: 130, Character: 140},
			},
		},
		{
			Dump:           store.Dump{ID: 44, RepositoryID: 50},
			Path:           "p2.go",
			AdjustedCommit: "deadbeef2",
			AdjustedRange: bundles.Range{
				Start: bundles.Position{Line: 210, Character: 220},
				End:   bundles.Position{Line: 230, Character: 240},
			},
		},
		{
			Dump:           store.Dump{ID: 44, RepositoryID: 50},
			Path:           "p3.go",
			AdjustedCommit: "deadbeef2",
			AdjustedRange: bundles.Range{
				Start: bundles.Position{Line: 310, Character: 320},
				End:   bundles.Position{Line: 330, Character: 340},
			},
		},
	}
	if diff := cmp.Diff(expectedDefinitions, definitions); diff != "" {
		t.Errorf("unexpected definitions (-want +got):\n%s", diff)
	}
}

func TestReferences(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockBundleManagerClient := bundlemocks.NewMockBundleManagerClient()
	mockBundleClient := bundlemocks.NewMockBundleClient()
	mockCodeIntelAPI := apimocks.NewMockCodeIntelAPI()
	mockPositionAdjuster := NewMockPositionAdjuster()

	testMoniker1 := bundles.MonikerData{Kind: "import", Scheme: "gomod", Identifier: "pad", PackageInformationID: "1234"}
	testMoniker2 := bundles.MonikerData{Kind: "export", Scheme: "gomod", Identifier: "pad", PackageInformationID: "1234"}

	// Cursor decoding
	mockStore.GetDumpByIDFunc.SetDefaultHook(func(ctx context.Context, id int) (store.Dump, bool, error) { return store.Dump{ID: id}, true, nil })
	mockBundleManagerClient.BundleClientFunc.SetDefaultReturn(mockBundleClient)
	mockBundleClient.MonikersByPositionFunc.SetDefaultReturn([][]bundles.MonikerData{{testMoniker1, testMoniker2}}, nil)

	// position can be translated for subsequent dumps
	mockPositionAdjuster.AdjustPositionFunc.SetDefaultReturn("", bundles.Position{Line: 20, Character: 15}, true, nil)

	// first requested dump (dump 42) has no equivalent position
	mockPositionAdjuster.AdjustPositionFunc.PushReturn("", bundles.Position{}, false, nil)

	// default behavior is empty result set
	mockCodeIntelAPI.ReferencesFunc.SetDefaultReturn(nil, codeintelapi.Cursor{}, false, nil)

	cursorIn1 := codeintelapi.Cursor{Phase: "p1"}
	cursorOut1 := codeintelapi.Cursor{Phase: "p2"}

	// first requested dump (dump 43) returns partial references
	mockCodeIntelAPI.ReferencesFunc.PushReturn([]codeintelapi.ResolvedLocation{
		{
			Dump: store.Dump{ID: 43, RepositoryID: 50},
			Path: "p1.go",
			Range: bundles.Range{
				Start: bundles.Position{Line: 11, Character: 12},
				End:   bundles.Position{Line: 13, Character: 14},
			},
		},
	}, cursorOut1, true, nil)

	cursorIn2 := codeintelapi.Cursor{Phase: "p3"}

	// second requested dump (dump 44) returns partial references
	mockCodeIntelAPI.ReferencesFunc.PushReturn([]codeintelapi.ResolvedLocation{
		{
			Dump: store.Dump{ID: 44, RepositoryID: 50},
			Path: "p2.go",
			Range: bundles.Range{
				Start: bundles.Position{Line: 21, Character: 22},
				End:   bundles.Position{Line: 23, Character: 24},
			},
		},
	}, codeintelapi.Cursor{}, false, nil)

	cursorIn3 := codeintelapi.Cursor{Phase: "p4"}
	cursorOut3 := codeintelapi.Cursor{Phase: "p5"}

	// third requested dump (dump 46) returns partial references
	mockCodeIntelAPI.ReferencesFunc.PushReturn([]codeintelapi.ResolvedLocation{
		{
			Dump: store.Dump{ID: 46, RepositoryID: 50},
			Path: "p3.go",
			Range: bundles.Range{
				Start: bundles.Position{Line: 31, Character: 32},
				End:   bundles.Position{Line: 33, Character: 34},
			},
		},
	}, cursorOut3, true, nil)

	mockPositionAdjuster.AdjustRangeFunc.SetDefaultHook(func(ctx context.Context, path, commit string, r bundles.Range, reverse bool) (string, bundles.Range, bool, error) {
		return path, bundles.Range{
			Start: bundles.Position{Line: r.Start.Line * 10, Character: r.Start.Character * 10},
			End:   bundles.Position{Line: r.End.Line * 10, Character: r.End.Character * 10},
		}, true, nil
	})

	queryResolver := NewQueryResolver(
		mockStore,
		mockBundleManagerClient,
		mockCodeIntelAPI,
		mockPositionAdjuster,
		50,
		"deadbeef2",
		"/foo/bar.go",
		[]store.Dump{
			{ID: 42, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 43, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 44, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 45, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 46, RepositoryID: 50, Commit: "deadbeef1"},
		},
	)

	cursor, err := makeCursor(map[int]string{
		42: "",
		43: codeintelapi.EncodeCursor(cursorIn1),
		44: codeintelapi.EncodeCursor(cursorIn2),
		46: codeintelapi.EncodeCursor(cursorIn3),
		47: "",
	})
	if err != nil {
		t.Fatalf("unexpected error creating cursor: %s", err)
	}

	references, nextCursor, err := queryResolver.References(context.Background(), 10, 15, 3, cursor)
	if err != nil {
		t.Fatalf("unexpected error resolving references: %s", err)
	}

	expectedCursor, err := makeCursor(map[int]string{
		43: codeintelapi.EncodeCursor(cursorOut1),
		46: codeintelapi.EncodeCursor(cursorOut3),
	})
	if err != nil {
		t.Fatalf("unexpected error creating cursor: %s", err)
	}

	if nextCursor != expectedCursor {
		t.Errorf("unexpected cursor. want=%q have=%q", expectedCursor, nextCursor)
	}

	expectedReferences := []AdjustedLocation{
		{
			Dump:           store.Dump{ID: 43, RepositoryID: 50},
			Path:           "p1.go",
			AdjustedCommit: "deadbeef2",
			AdjustedRange: bundles.Range{
				Start: bundles.Position{Line: 110, Character: 120},
				End:   bundles.Position{Line: 130, Character: 140},
			},
		},
		{
			Dump:           store.Dump{ID: 44, RepositoryID: 50},
			Path:           "p2.go",
			AdjustedCommit: "deadbeef2",
			AdjustedRange: bundles.Range{
				Start: bundles.Position{Line: 210, Character: 220},
				End:   bundles.Position{Line: 230, Character: 240},
			},
		},
		{
			Dump:           store.Dump{ID: 46, RepositoryID: 50},
			Path:           "p3.go",
			AdjustedCommit: "deadbeef2",
			AdjustedRange: bundles.Range{
				Start: bundles.Position{Line: 310, Character: 320},
				End:   bundles.Position{Line: 330, Character: 340},
			},
		},
	}
	if diff := cmp.Diff(expectedReferences, references); diff != "" {
		t.Errorf("unexpected references (-want +got):\n%s", diff)
	}

	if val := len(mockCodeIntelAPI.ReferencesFunc.History()); val != 3 {
		t.Errorf("unexpected call count. want=%d have=%d", 3, val)
	}
	if val := mockCodeIntelAPI.ReferencesFunc.History()[0].Arg3; val != 3 {
		t.Errorf("unexpected limit. want=%d have=%d", 3, val)
	}
	if diff := cmp.Diff(cursorIn1, mockCodeIntelAPI.ReferencesFunc.History()[0].Arg4); diff != "" {
		t.Errorf("unexpected cursor (-want +got):\n%s", diff)
	}
	if val := mockCodeIntelAPI.ReferencesFunc.History()[1].Arg3; val != 3 {
		t.Errorf("unexpected limit. want=%d have=%d", 3, val)
	}
	if diff := cmp.Diff(cursorIn2, mockCodeIntelAPI.ReferencesFunc.History()[1].Arg4); diff != "" {
		t.Errorf("unexpected cursor (-want +got):\n%s", diff)
	}
	if val := mockCodeIntelAPI.ReferencesFunc.History()[2].Arg3; val != 3 {
		t.Errorf("unexpected limit. want=%d have=%d", 3, val)
	}
	if diff := cmp.Diff(cursorIn3, mockCodeIntelAPI.ReferencesFunc.History()[2].Arg4); diff != "" {
		t.Errorf("unexpected cursor (-want +got):\n%s", diff)
	}
}

func TestHover(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockBundleManagerClient := bundlemocks.NewMockBundleManagerClient()
	mockCodeIntelAPI := apimocks.NewMockCodeIntelAPI()
	mockPositionAdjuster := NewMockPositionAdjuster()

	// position can be translated for subsequent dumps
	mockPositionAdjuster.AdjustPositionFunc.SetDefaultReturn("", bundles.Position{Line: 20, Character: 15}, true, nil)

	// first requested dump (dump 42) has no equivalent position
	mockPositionAdjuster.AdjustPositionFunc.PushReturn("", bundles.Position{}, false, nil)

	mockCodeIntelAPI.HoverFunc.SetDefaultReturn("hover text", bundles.Range{
		Start: bundles.Position{Line: 11, Character: 12},
		End:   bundles.Position{Line: 13, Character: 14},
	}, true, nil)

	// first requested dump (dump 43) has no defined hover
	mockCodeIntelAPI.HoverFunc.PushReturn("", bundles.Range{}, false, nil)

	mockPositionAdjuster.AdjustRangeFunc.SetDefaultHook(func(ctx context.Context, path, commit string, r bundles.Range, reverse bool) (string, bundles.Range, bool, error) {
		return path, bundles.Range{
			Start: bundles.Position{Line: r.Start.Line * 10, Character: r.Start.Character * 10},
			End:   bundles.Position{Line: r.End.Line * 10, Character: r.End.Character * 10},
		}, true, nil
	})

	queryResolver := NewQueryResolver(
		mockStore,
		mockBundleManagerClient,
		mockCodeIntelAPI,
		mockPositionAdjuster,
		50,
		"deadbeef2",
		"/foo/bar.go",
		[]store.Dump{
			{ID: 42, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 43, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 44, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 45, RepositoryID: 50, Commit: "deadbeef1"},
		},
	)

	text, r, ok, err := queryResolver.Hover(context.Background(), 10, 15)
	if err != nil {
		t.Fatalf("unexpected error resolving hover: %s", err)
	}
	if !ok {
		t.Fatalf("expected hover text")
	}

	if text != "hover text" {
		t.Errorf("unexpected text. want=%q have=%q", "hover text", text)
	}

	expectedRange := bundles.Range{
		Start: bundles.Position{Line: 110, Character: 120},
		End:   bundles.Position{Line: 130, Character: 140},
	}
	if diff := cmp.Diff(expectedRange, r); diff != "" {
		t.Errorf("unexpected range (-want +got):\n%s", diff)
	}
}

func TestDiagnostics(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockBundleManagerClient := bundlemocks.NewMockBundleManagerClient()
	mockCodeIntelAPI := apimocks.NewMockCodeIntelAPI()
	mockPositionAdjuster := NewMockPositionAdjuster()

	// position can be translated for subsequent dumps
	mockPositionAdjuster.AdjustPathFunc.SetDefaultReturn("/foo/bar.go", true, nil)

	// first requested dump (dump 42) has no equivalent path
	mockPositionAdjuster.AdjustPathFunc.PushReturn("", false, nil)

	// first requested dump (dump 43) returns partial diagnostics
	mockCodeIntelAPI.DiagnosticsFunc.PushReturn([]codeintelapi.ResolvedDiagnostic{
		{
			Dump: store.Dump{ID: 43, RepositoryID: 50},
			Diagnostic: bundles.Diagnostic{
				Path:           "p1",
				Severity:       1,
				Code:           "c1",
				Message:        "m1",
				Source:         "s1",
				StartLine:      11,
				StartCharacter: 12,
				EndLine:        13,
				EndCharacter:   14,
			},
		},
	}, 1, nil)

	// second requested dump (dump 44) returns partial diagnostics
	mockCodeIntelAPI.DiagnosticsFunc.PushReturn([]codeintelapi.ResolvedDiagnostic{
		{
			Dump: store.Dump{ID: 44, RepositoryID: 50},
			Diagnostic: bundles.Diagnostic{
				Path:           "p2",
				Severity:       2,
				Code:           "c2",
				Message:        "m2",
				Source:         "s2",
				StartLine:      21,
				StartCharacter: 22,
				EndLine:        23,
				EndCharacter:   24,
			},
		},
		{
			Dump: store.Dump{ID: 44, RepositoryID: 50},
			Diagnostic: bundles.Diagnostic{
				Path:           "p3",
				Severity:       3,
				Code:           "c3",
				Message:        "m3",
				Source:         "s3",
				StartLine:      31,
				StartCharacter: 32,
				EndLine:        33,
				EndCharacter:   34,
			},
		},
	}, 14, nil)

	// third requested dump (dump 45) returns only total count
	mockCodeIntelAPI.DiagnosticsFunc.SetDefaultReturn(nil, 3, nil)

	mockPositionAdjuster.AdjustRangeFunc.SetDefaultHook(func(ctx context.Context, path, commit string, r bundles.Range, reverse bool) (string, bundles.Range, bool, error) {
		return path, bundles.Range{
			Start: bundles.Position{Line: r.Start.Line * 10, Character: r.Start.Character * 10},
			End:   bundles.Position{Line: r.End.Line * 10, Character: r.End.Character * 10},
		}, true, nil
	})

	queryResolver := NewQueryResolver(
		mockStore,
		mockBundleManagerClient,
		mockCodeIntelAPI,
		mockPositionAdjuster,
		50,
		"deadbeef2",
		"/foo/bar.go",
		[]store.Dump{
			{ID: 42, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 43, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 44, RepositoryID: 50, Commit: "deadbeef1"},
			{ID: 45, RepositoryID: 50, Commit: "deadbeef1"},
		},
	)

	diagnostics, totalCount, err := queryResolver.Diagnostics(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error resolving diagnostics: %s", err)
	}

	if totalCount != 18 {
		t.Errorf("unexpected total count. want=%d have=%d", 18, totalCount)
	}

	expectedDiagnostics := []AdjustedDiagnostic{
		{
			Diagnostic: bundles.Diagnostic{
				Path:           "p1",
				Severity:       1,
				Code:           "c1",
				Message:        "m1",
				Source:         "s1",
				StartLine:      11,
				StartCharacter: 12,
				EndLine:        13,
				EndCharacter:   14,
			},
			Dump:           store.Dump{ID: 43, RepositoryID: 50},
			AdjustedCommit: "deadbeef2",
			AdjustedRange: bundles.Range{
				Start: bundles.Position{Line: 110, Character: 120},
				End:   bundles.Position{Line: 130, Character: 140},
			},
		},
		{
			Diagnostic: bundles.Diagnostic{
				Path:           "p2",
				Severity:       2,
				Code:           "c2",
				Message:        "m2",
				Source:         "s2",
				StartLine:      21,
				StartCharacter: 22,
				EndLine:        23,
				EndCharacter:   24,
			},
			Dump:           store.Dump{ID: 44, RepositoryID: 50},
			AdjustedCommit: "deadbeef2",
			AdjustedRange: bundles.Range{
				Start: bundles.Position{Line: 210, Character: 220},
				End:   bundles.Position{Line: 230, Character: 240},
			},
		},
		{
			Diagnostic: bundles.Diagnostic{
				Path:           "p3",
				Severity:       3,
				Code:           "c3",
				Message:        "m3",
				Source:         "s3",
				StartLine:      31,
				StartCharacter: 32,
				EndLine:        33,
				EndCharacter:   34,
			},
			Dump:           store.Dump{ID: 44, RepositoryID: 50},
			AdjustedCommit: "deadbeef2",
			AdjustedRange: bundles.Range{
				Start: bundles.Position{Line: 310, Character: 320},
				End:   bundles.Position{Line: 330, Character: 340},
			},
		},
	}
	if diff := cmp.Diff(expectedDiagnostics, diagnostics); diff != "" {
		t.Errorf("unexpected diagnostics (-want +got):\n%s", diff)
	}

	if val := len(mockCodeIntelAPI.DiagnosticsFunc.History()); val != 3 {
		t.Errorf("unexpected call count. want=%d have=%d", 3, val)
	}
	if val := mockCodeIntelAPI.DiagnosticsFunc.History()[0].Arg3; val != 3 {
		t.Errorf("unexpected limit. want=%d have=%d", 3, val)
	}
	if val := mockCodeIntelAPI.DiagnosticsFunc.History()[1].Arg3; val != 2 {
		t.Errorf("unexpected limit. want=%d have=%d", 2, val)
	}
	if val := mockCodeIntelAPI.DiagnosticsFunc.History()[2].Arg3; val != 0 {
		t.Errorf("unexpected limit. want=%d have=%d", 0, val)
	}
}
