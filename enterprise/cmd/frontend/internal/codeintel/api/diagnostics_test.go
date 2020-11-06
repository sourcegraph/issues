package api

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	store "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	storemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore/mocks"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/lsifstore"
	bundlemocks "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/lsifstore/mocks"
)

func TestDiagnostics(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockBundleStore := bundlemocks.NewMockStore()
	mockGitserverClient := NewMockGitserverClient()

	sourceDiagnostics := []lsifstore.Diagnostic{
		{
			Path: "internal/foo.go",
			DiagnosticData: lsifstore.DiagnosticData{
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
		{
			Path: "internal/bar.go",
			DiagnosticData: lsifstore.DiagnosticData{
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
			Path: "internal/baz.go",
			DiagnosticData: lsifstore.DiagnosticData{
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
	}

	setMockStoreGetDumpByID(t, mockStore, map[int]store.Dump{42: testDump1})
	setMockBundleStoreDiagnostics(t, mockBundleStore, 42, "sub1", 1, 3, sourceDiagnostics, 5)

	api := testAPI(mockStore, mockBundleStore, mockGitserverClient)
	diagnostics, _, err := api.Diagnostics(context.Background(), "sub1", 42, 3, 1)
	if err != nil {
		t.Fatalf("expected error getting diagnostics: %s", err)
	}

	expectedDiagnostics := []ResolvedDiagnostic{
		{
			Dump: store.Dump{
				ID:   42,
				Root: "sub1/",
			},
			Diagnostic: lsifstore.Diagnostic{
				Path: "sub1/internal/foo.go",
				DiagnosticData: lsifstore.DiagnosticData{
					Severity:       1,
					Code:           "c1",
					Message:        "m1",
					Source:         "s1",
					StartLine:      11,
					StartCharacter: 12,
					EndLine:        13,
					EndCharacter:   14,
				}},
		},
		{
			Dump: store.Dump{
				ID:   42,
				Root: "sub1/",
			},
			Diagnostic: lsifstore.Diagnostic{
				Path: "sub1/internal/bar.go",
				DiagnosticData: lsifstore.DiagnosticData{
					Severity:       2,
					Code:           "c2",
					Message:        "m2",
					Source:         "s2",
					StartLine:      21,
					StartCharacter: 22,
					EndLine:        23,
					EndCharacter:   24,
				}},
		},
		{
			Dump: store.Dump{
				ID:   42,
				Root: "sub1/",
			},
			Diagnostic: lsifstore.Diagnostic{
				Path: "sub1/internal/baz.go",
				DiagnosticData: lsifstore.DiagnosticData{
					Severity:       3,
					Code:           "c3",
					Message:        "m3",
					Source:         "s3",
					StartLine:      31,
					StartCharacter: 32,
					EndLine:        33,
					EndCharacter:   34,
				}},
		},
	}
	if diff := cmp.Diff(expectedDiagnostics, diagnostics); diff != "" {
		t.Errorf("unexpected range (-want +got):\n%s", diff)
	}
}

func TestDiagnosticsUnknownDump(t *testing.T) {
	mockStore := storemocks.NewMockStore()
	mockBundleStore := bundlemocks.NewMockStore()
	mockGitserverClient := NewMockGitserverClient()
	setMockStoreGetDumpByID(t, mockStore, nil)

	api := testAPI(mockStore, mockBundleStore, mockGitserverClient)
	if _, _, err := api.Diagnostics(context.Background(), "sub1", 42, 0, 10); err != ErrMissingDump {
		t.Fatalf("unexpected error getting diagnostics. want=%q have=%q", ErrMissingDump, err)
	}
}
