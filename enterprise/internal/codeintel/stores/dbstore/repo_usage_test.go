package dbstore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
)

func TestRepoUsageStatistics(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	store := testStore(db)

	insertEvent := func(name string, count, repoID int) {
		json := fmt.Sprintf(`{"repositoryId": %d}`, repoID)
		query := sqlf.Sprintf(`
			INSERT INTO event_logs (user_id, anonymous_user_id, source, argument, version, timestamp, name, url)
			VALUES (1, '', 'test', %s, 'dev', NOW(), %s, '')
		`, json, name)

		for i := 0; i < count; i++ {
			if _, err := db.Exec(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
				t.Fatalf("unexpected error inserting event record: %s", err)
			}
		}
	}

	for _, data := range []struct {
		RepoID           int
		NumSearchEvents  int
		NumPreciseEvents int
	}{
		{1, 10, 10},
		{2, 25, 20},
		{3, 15, 30},
		{4, 5, 40},
		{5, 4, 50},
		{6, 10, 60}, // deleted repo
		{7, 10, 60}, // no such repo
	} {
		insertEvent("codeintel.searchHover", data.NumSearchEvents, data.RepoID)
		insertEvent("codeintel.lsifHover", data.NumPreciseEvents, data.RepoID)
	}

	repos := []string{
		"github.com/foo/baz",
		"github.com/foo/bar",
		"gitlab.com/bar/baz",
		"github.com/bar/bonk",
		"github.com/bonk/quux",
	}
	for i, name := range repos {
		query := sqlf.Sprintf(`INSERT INTO repo (id, name, uri) VALUES (%s, %s, %s)`, i+1, name, name)

		if _, err := db.Exec(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
			t.Fatalf("unexpected error inserting repo: %s", err)
		}
	}

	query := sqlf.Sprintf(
		`INSERT INTO repo (id, name, uri, deleted_at) VALUES (%s, %s, %s, %s)`,
		len(repos)+1,
		"DELETED-github.com/baz/honk",
		"github.com/baz/honk",
		time.Now(),
	)
	if _, err := db.Exec(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
		t.Fatalf("unexpected error inserting repo: %s", err)
	}

	stats, err := store.RepoUsageStatistics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting repo counts: %s", err)
	}

	expectedStatistics := []RepoUsageStatistics{
		{RepositoryID: 2, SearchCount: 25, PreciseCount: 20},
		{RepositoryID: 3, SearchCount: 15, PreciseCount: 30},
		{RepositoryID: 1, SearchCount: 10, PreciseCount: 10},
		{RepositoryID: 4, SearchCount: 5, PreciseCount: 40},
		{RepositoryID: 5, SearchCount: 4, PreciseCount: 50},
	}
	if diff := cmp.Diff(expectedStatistics, stats); diff != "" {
		t.Errorf("unexpected repo counts (-want +got):\n%s", diff)
	}
}
