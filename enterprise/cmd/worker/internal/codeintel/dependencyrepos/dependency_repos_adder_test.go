package dependencyrepos

import (
	"context"
	"testing"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/lsifstore"
)

func TestDependencyReposAddingHandlerJVM(t *testing.T) {
	t.Run("successful handle", func(t *testing.T) {
		mockDBStore := NewMockDBStore()
		mockScanner := NewMockPackageReferenceScanner()
		mockDBStore.ReferencesForUploadFunc.SetDefaultReturn(mockScanner, nil)
		mockScanner.NextFunc.PushReturn(lsifstore.PackageReference{
			Package: lsifstore.Package{
				DumpID:  5,
				Scheme:  "semanticdb",
				Name:    "maven/com.google.guava/guava/29.0-jre",
				Version: "29.0-jre",
			},
		}, true, nil)
		mockScanner.NextFunc.PushReturn(lsifstore.PackageReference{
			Package: lsifstore.Package{
				DumpID:  5,
				Scheme:  "semanticdb",
				Name:    "maven/jdk/8",
				Version: "8",
			},
		}, true, nil)

		handler := &dependencyRepoAddingHandler{
			dbStore: mockDBStore,
		}

		job := dbstore.DependencyRepoAddingJob{
			UploadID: 5,
		}

		if err := handler.Handle(context.Background(), job); err != nil {
			t.Fatalf("unexpected error adding dependency repo: %s", err)
		}

		if len(mockDBStore.InsertCloneableDependencyReposFunc.History()) != 1 {
			t.Errorf("unexpected number of inserts. wanted=%d, got=%d\n", 1, len(mockDBStore.InsertCloneableDependencyReposFunc.History()))
		}
	})

	t.Run("unknown reference schema", func(t *testing.T) {
		mockDBStore := NewMockDBStore()
		mockScanner := NewMockPackageReferenceScanner()
		mockDBStore.ReferencesForUploadFunc.SetDefaultReturn(mockScanner, nil)
		mockScanner.NextFunc.PushReturn(lsifstore.PackageReference{
			Package: lsifstore.Package{
				DumpID:  5,
				Scheme:  "banana",
				Name:    "sampletext",
				Version: "421",
			},
		}, true, nil)

		handler := &dependencyRepoAddingHandler{
			dbStore: mockDBStore,
		}

		job := dbstore.DependencyRepoAddingJob{
			UploadID: 5,
		}

		schemaErr := &ReferenceSchemeError{}
		err := handler.Handle(context.Background(), job)
		if err == nil {
			t.Fatalf("expected error, got none")
		}
		if !errors.As(err, &schemaErr) {
			t.Fatalf("unexpected error adding dependency repo, wanted=%T, got=%T %v", schemaErr, err, err)
		}
	})
}
