package db

import (
	"os"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

func TestMigrations(t *testing.T) {
	if os.Getenv("SKIP_MIGRATION_TEST") != "" {
		t.Skip()
	}

	// Setup a global test database
	dbtesting.SetupGlobalTestDB(t)

	migrate := func() {
		for _, databaseName := range dbutil.DatabaseNames {
			if err := dbconn.MigrateDB(dbconn.Global, databaseName); err != nil {
				t.Errorf("error running initial migrations: %s", err)
			}
		}
	}

	for _, databaseName := range dbutil.DatabaseNames {
		t.Run(databaseName, func(t *testing.T) {
			// Dropping a squash schema _all_ the way down just drops the entire public
			// schema. Because we have a "combined" database that runs migrations for
			// multiple disjoint schemas in development environments, migrating all the
			// way down will drop all tables from all schemas. This loop runs such down
			// migrations, so we prep our tests by re-migrating up on each iteration.
			migrate()

			m, err := dbutil.NewMigrate(dbconn.Global, databaseName)
			if err != nil {
				t.Errorf("error constructing migrations: %s", err)
			}
			// Run all down migrations then up migrations again to ensure there are no SQL errors.
			if err := m.Down(); err != nil {
				t.Errorf("error running down migrations: %s", err)
			}
			if err := dbutil.DoMigrate(m); err != nil {
				t.Errorf("error running up migrations: %s", err)
			}
		})
	}
}

func TestPassword(t *testing.T) {
	// By default we use fast mocks for our password in tests. This ensures
	// our actual implementation is correct.
	oldHash := dbtesting.MockHashPassword
	oldValid := dbtesting.MockValidPassword
	dbtesting.MockHashPassword = nil
	dbtesting.MockValidPassword = nil
	defer func() {
		dbtesting.MockHashPassword = oldHash
		dbtesting.MockValidPassword = oldValid
	}()

	h, err := hashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	if !validPassword(h.String, "correct-password") {
		t.Fatal("validPassword should of returned true")
	}
	if validPassword(h.String, "wrong-password") {
		t.Fatal("validPassword should of returned false")
	}
}
