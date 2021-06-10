package dbstore

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
)

func TestRepoName(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	store := testStore(db)

	if _, err := db.Exec(`INSERT INTO repo (id, name) VALUES (50, 'github.com/foo/bar')`); err != nil {
		t.Fatalf("unexpected error inserting repo: %s", err)
	}

	name, err := store.RepoName(context.Background(), 50)
	if err != nil {
		t.Fatalf("unexpected error getting repo name: %s", err)
	}
	if name != "github.com/foo/bar" {
		t.Errorf("unexpected repo name. want=%s have=%s", "github.com/foo/bar", name)
	}
}
