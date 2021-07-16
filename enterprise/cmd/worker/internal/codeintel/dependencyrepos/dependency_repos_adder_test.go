package dependencyrepos

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/lsifstore"
)

func TestDependencyReposAddingHandlerJVM(t *testing.T) {
	mockDBStore := NewMockDBStore()
	mockScanner := NewMockPackageReferenceScanner()
	mockDBStore.WithFunc.SetDefaultReturn(mockDBStore)
	mockDBStore.ReferencesForUploadFunc.SetDefaultReturn(mockScanner, nil)
	mockScanner.NextFunc.PushReturn(lsifstore.PackageReference{
		Package: lsifstore.Package{
			DumpID:  5,
			Scheme:  "semanticdb",
			Name:    "maven/com.google.guava/guava/29.0-jre",
			Version: "29.0-jre",
		},
	}, true, nil)

	handler := &dependencyRepoAddingHandler{
		dbStore: mockDBStore,
	}

	job := dbstore.DependencyRepoAddingJob{
		UploadID: 5,
	}

	if err := handler.Handle(context.Background(), nil, job); err != nil {
		t.Fatalf("unexpected error adding dependency repo: %s", err)
	}
}
