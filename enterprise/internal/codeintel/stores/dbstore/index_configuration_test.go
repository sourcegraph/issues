package dbstore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
)

func TestGetRepositoriesWithIndexConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtesting.GetDB(t)
	store := testStore(db)

	for _, repositoryID := range []int{42, 43, 44, 45, 46} {
		var deletedAt *time.Time
		if repositoryID == 46 {
			t := time.Now()
			deletedAt = &t
		}

		query := sqlf.Sprintf(
			`INSERT INTO repo (id, name, deleted_at) VALUES (%s, %s, %s)`,
			repositoryID,
			fmt.Sprintf("github.com/baz/honk%2d", repositoryID),
			deletedAt,
		)
		if _, err := db.Exec(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
			t.Fatalf("unexpected error inserting repo: %s", err)
		}
	}

	for i, repositoryID := range []int{42, 44, 45, 46} {
		query := sqlf.Sprintf(
			`INSERT INTO lsif_index_configuration (id, repository_id, data) VALUES (%s, %s, %s)`,
			i,
			repositoryID,
			[]byte(`test`),
		)
		if _, err := db.Exec(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
			t.Fatalf("unexpected error inserting repo: %s", err)
		}
	}

	repositoryIDs, err := store.GetRepositoriesWithIndexConfiguration(context.Background())
	if err != nil {
		t.Fatalf("unexpected error while fetching repositories with index configuration: %s", err)
	}

	expectedRepositoryIDs := []int{
		42,
		44,
		45,
	}
	if diff := cmp.Diff(expectedRepositoryIDs, repositoryIDs); diff != "" {
		t.Errorf("unexpected repository identifiers (-want +got):\n%s", diff)
	}
}

func TestGetIndexConfigurationByRepositoryID(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtesting.GetDB(t)
	store := testStore(db)

	expectedConfigurationData := []byte(`{
		"foo": "bar",
		"baz": "bonk",
	}`)

	query := sqlf.Sprintf(
		`INSERT INTO repo (id, name) VALUES (%s, %s)`,
		42,
		"github.com/baz/honk",
	)
	if _, err := db.Exec(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
		t.Fatalf("unexpected error inserting repo: %s", err)
	}

	query = sqlf.Sprintf(
		`INSERT INTO lsif_index_configuration (id, repository_id, data) VALUES (%s, %s, %s)`,
		1,
		42,
		expectedConfigurationData,
	)
	if _, err := db.Exec(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
		t.Fatalf("unexpected error inserting repo: %s", err)
	}

	indexConfiguration, ok, err := store.GetIndexConfigurationByRepositoryID(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error while fetching index configuration: %s", err)
	}
	if !ok {
		t.Fatalf("expected a configuration record")
	}

	if diff := cmp.Diff(expectedConfigurationData, indexConfiguration.Data); diff != "" {
		t.Errorf("unexpected configuration payload (-want +got):\n%s", diff)
	}
}

func TestGetRepositoriesWithIndexConfigurationIgnoresDisabledRepos(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtesting.GetDB(t)
	store := testStore(db)

	for _, repositoryID := range []int{42, 43, 44, 45, 46} {
		var deletedAt *time.Time
		if repositoryID == 46 {
			t := time.Now()
			deletedAt = &t
		}

		query := sqlf.Sprintf(
			`INSERT INTO repo (id, name, deleted_at) VALUES (%s, %s, %s)`,
			repositoryID,
			fmt.Sprintf("github.com/baz/honk%2d", repositoryID),
			deletedAt,
		)
		if _, err := db.Exec(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
			t.Fatalf("unexpected error inserting repo: %s", err)
		}
	}

	// Only even repos are enabled
	for i, repositoryID := range []int{42, 44, 45, 46} {
		query := sqlf.Sprintf(
			`INSERT INTO lsif_index_configuration (id, repository_id, autoindex_enabled, data) VALUES (%s, %s, %s, %s)`,
			i,
			repositoryID,
			repositoryID%2 == 0,
			[]byte(`test`),
		)
		if _, err := db.Exec(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
			t.Fatalf("unexpected error inserting repo: %s", err)
		}
	}

	repositoryIDs, err := store.GetRepositoriesWithIndexConfiguration(context.Background())
	if err != nil {
		t.Fatalf("unexpected error while fetching repositories with index configuration: %s", err)
	}

	expectedRepositoryIDs := []int{
		42,
		44,
	}
	if diff := cmp.Diff(expectedRepositoryIDs, repositoryIDs); diff != "" {
		t.Errorf("unexpected repository identifiers (-want +got):\n%s", diff)
	}

	disabledRepositoryIDs, err := store.GetAutoindexDisabledRepositories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting disabled repositories: %s", err)
	}

	expectedDisabledRepositoryIDs := []int{45}
	if diff := cmp.Diff(expectedDisabledRepositoryIDs, disabledRepositoryIDs); diff != "" {
		t.Errorf("unexpected repository identifiers (-want +got):\n%s", diff)
	}
}
