package migrations_test

import (
	"database/sql"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"

	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/migrations"
)

func TestIDConstraints(t *testing.T) {
	cases := []struct {
		Name string
		FS   fs.FS
	}{
		{Name: "frontend", FS: migrations.Frontend},
		{Name: "codeintel", FS: migrations.CodeIntel},
		{Name: "codeinsights", FS: migrations.CodeInsights},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ups, err := fs.Glob(c.FS, "*.up.sql")
			if err != nil {
				t.Fatal(err)
			}

			if len(ups) == 0 {
				t.Fatal("no up migrations found")
			}

			byID := map[int][]string{}
			for _, name := range ups {
				id, err := strconv.Atoi(name[:strings.IndexByte(name, '_')])
				if err != nil {
					t.Fatalf("failed to parse name %q: %v", name, err)
				}
				byID[id] = append(byID[id], name)
			}

			var ids []int
			for id, names := range byID {
				if len(names) > 1 {
					t.Errorf("multiple migrations with ID %d: %s", id, strings.Join(names, " "))
				}

				ids = append(ids, id)
			}
			sort.Ints(ids)

			for i, id := range ids {
				if i != 0 && ids[i-1]+1 != id {
					// Check if we are using sequential migrations.
					t.Errorf("gap in migrations between %s and %s", byID[ids[i-1]][0], byID[id][0])
				}
			}
		})
	}
}

func TestFrontendMigrations(t *testing.T) {
	if os.Getenv("SKIP_MIGRATION_TEST") != "" {
		t.Skip()
	}

	// Setup a global test database
	db := dbtest.NewDB(t, "")
	testMigrations(t, db, dbconn.Frontend)
}

func TestCodeIntelMigrations(t *testing.T) {
	if os.Getenv("SKIP_MIGRATION_TEST") != "" {
		t.Skip()
	}

	// Setup a global test database
	db := dbtest.NewDB(t, "")
	testMigrations(t, db, dbconn.CodeIntel)
}

// testMigrations runs all migrations up, then the migrations for the given database
// all the way back down, then back up to check for syntax errors and reversibility.
func testMigrations(t *testing.T, db *sql.DB, database *dbconn.Database) {
	m, err := dbconn.NewMigrate(db, database)
	if err != nil {
		t.Errorf("error constructing migrations: %s", err)
	}

	for _, database := range []*dbconn.Database{
		dbconn.Frontend,
		dbconn.CodeIntel,
	} {
		if err := dbconn.MigrateDB(db, database); err != nil {
			t.Errorf("unexpected error running initial migrations: %s", err)
		}
	}

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		t.Errorf("unexpected error running down migrations: %s", err)
	}
	if _, err := db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
		t.Fatalf("failed to recreate schema")
	}

	m, err = dbconn.NewMigrate(db, database)
	if err != nil {
		t.Errorf("unexpected error constructing migrations: %s", err)
	}
	if err := m.Up(); err != nil {
		t.Errorf("unexpected error re-running up migrations: %s", err)
	}
}
