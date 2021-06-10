package oobmigration

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

func TestList(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	store := testStore(t, db)

	migrations, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting migrations: %s", err)
	}

	expectedMigrations := make([]Migration, len(testMigrations))
	copy(expectedMigrations, testMigrations)
	sort.Slice(expectedMigrations, func(i, j int) bool {
		return expectedMigrations[i].ID > expectedMigrations[j].ID
	})

	if diff := cmp.Diff(expectedMigrations, migrations); diff != "" {
		t.Errorf("unexpected migrations (-want +got):\n%s", diff)
	}
}

func TestListEnterprise(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	store := testStore(t, db)

	ReturnEnterpriseMigrations = true
	defer func() { ReturnEnterpriseMigrations = false }()

	migrations, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting migrations: %s", err)
	}

	expectedMigrations := make([]Migration, len(testMigrations))
	copy(expectedMigrations, testMigrations)
	expectedMigrations = append(expectedMigrations, testEnterpriseMigrations...)
	sort.Slice(expectedMigrations, func(i, j int) bool {
		return expectedMigrations[i].ID > expectedMigrations[j].ID
	})

	if diff := cmp.Diff(expectedMigrations, migrations); diff != "" {
		t.Errorf("unexpected migrations (-want +got):\n%s", diff)
	}
}

func TestUpdateDirection(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	store := testStore(t, db)

	if err := store.UpdateDirection(context.Background(), 3, true); err != nil {
		t.Fatalf("unexpected error updating direction: %s", err)
	}

	migration, exists, err := store.GetByID(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error getting migrations: %s", err)
	}
	if !exists {
		t.Fatalf("expected record to exist")
	}

	expectedMigration := testMigrations[2] // ID = 3
	expectedMigration.ApplyReverse = true

	if diff := cmp.Diff(expectedMigration, migration); diff != "" {
		t.Errorf("unexpected migration (-want +got):\n%s", diff)
	}
}

func TestUpdateProgress(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	now := testTime.Add(time.Hour * 7)
	db := dbtest.NewDB(t, "")
	store := testStore(t, db)

	if err := store.updateProgress(context.Background(), 3, 0.7, now); err != nil {
		t.Fatalf("unexpected error updating migration: %s", err)
	}

	migration, exists, err := store.GetByID(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error getting migrations: %s", err)
	}
	if !exists {
		t.Fatalf("expected record to exist")
	}

	expectedMigration := testMigrations[2] // ID = 3
	expectedMigration.Progress = 0.7
	expectedMigration.LastUpdated = timePtr(now)

	if diff := cmp.Diff(expectedMigration, migration); diff != "" {
		t.Errorf("unexpected migration (-want +got):\n%s", diff)
	}
}

func TestAddError(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	now := testTime.Add(time.Hour * 8)
	db := dbtest.NewDB(t, "")
	store := testStore(t, db)

	if err := store.addError(context.Background(), 2, "oops", now); err != nil {
		t.Fatalf("unexpected error updating migration: %s", err)
	}

	migration, exists, err := store.GetByID(context.Background(), 2)
	if err != nil {
		t.Fatalf("unexpected error getting migrations: %s", err)
	}
	if !exists {
		t.Fatalf("expected record to exist")
	}

	expectedMigration := testMigrations[1] // ID = 2
	expectedMigration.LastUpdated = timePtr(now)
	expectedMigration.Errors = []MigrationError{
		{Message: "oops", Created: now},
		{Message: "uh-oh 1", Created: testTime.Add(time.Hour*5 + time.Second*2)},
		{Message: "uh-oh 2", Created: testTime.Add(time.Hour*5 + time.Second*1)},
	}

	if diff := cmp.Diff(expectedMigration, migration); diff != "" {
		t.Errorf("unexpected migration (-want +got):\n%s", diff)
	}
}

func TestAddErrorBounded(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	now := testTime.Add(time.Hour * 9)
	db := dbtest.NewDB(t, "")
	store := testStore(t, db)

	var expectedErrors []MigrationError
	for i := 0; i < MaxMigrationErrors*1.5; i++ {
		now = now.Add(time.Second)

		if err := store.addError(context.Background(), 2, fmt.Sprintf("oops %d", i), now); err != nil {
			t.Fatalf("unexpected error updating migration: %s", err)
		}

		expectedErrors = append(expectedErrors, MigrationError{
			Message: fmt.Sprintf("oops %d", i),
			Created: now,
		})
	}

	migration, exists, err := store.GetByID(context.Background(), 2)
	if err != nil {
		t.Fatalf("unexpected error getting migration: %s", err)
	}
	if !exists {
		t.Fatalf("expected record to exist")
	}

	n := len(expectedErrors) - 1
	for i := 0; i < len(expectedErrors)/2; i++ {
		expectedErrors[i], expectedErrors[n-i] = expectedErrors[n-i], expectedErrors[i]
	}

	expectedMigration := testMigrations[1] // ID = 2
	expectedMigration.LastUpdated = timePtr(now)
	expectedMigration.Errors = expectedErrors[:MaxMigrationErrors]

	if diff := cmp.Diff(expectedMigration, migration); diff != "" {
		t.Errorf("unexpected migrations (-want +got):\n%s", diff)
	}
}

//
//

var testTime = time.Unix(1613414740, 0)

var testMigrations = []Migration{
	{
		ID:             1,
		Team:           "search",
		Component:      "zoekt-index",
		Description:    "rot13 all the indexes for security",
		Introduced:     NewVersion(3, 25),
		Deprecated:     nil,
		Progress:       0,
		Created:        testTime,
		LastUpdated:    nil,
		NonDestructive: false,
		ApplyReverse:   false,
		Errors:         []MigrationError{},
	},
	{
		ID:             2,
		Team:           "codeintel",
		Component:      "lsif_data_documents",
		Description:    "denormalize counts",
		Introduced:     NewVersion(3, 26),
		Deprecated:     newVersionPtr(3, 28),
		Progress:       0.5,
		Created:        testTime.Add(time.Hour * 1),
		LastUpdated:    timePtr(testTime.Add(time.Hour * 2)),
		NonDestructive: true,
		ApplyReverse:   false,
		Errors: []MigrationError{
			{Message: "uh-oh 1", Created: testTime.Add(time.Hour*5 + time.Second*2)},
			{Message: "uh-oh 2", Created: testTime.Add(time.Hour*5 + time.Second*1)},
		},
	},
	{
		ID:             3,
		Team:           "platform",
		Component:      "lsif_data_documents",
		Description:    "gzip payloads",
		Introduced:     NewVersion(3, 24),
		Deprecated:     nil,
		Progress:       0.4,
		Created:        testTime.Add(time.Hour * 3),
		LastUpdated:    timePtr(testTime.Add(time.Hour * 4)),
		NonDestructive: false,
		ApplyReverse:   true,
		Errors: []MigrationError{
			{Message: "uh-oh 3", Created: testTime.Add(time.Hour*5 + time.Second*4)},
			{Message: "uh-oh 4", Created: testTime.Add(time.Hour*5 + time.Second*3)},
		},
	},
}

var testEnterpriseMigrations = []Migration{
	{
		ID:             4,
		Team:           "search",
		Component:      "zoekt-index",
		Description:    "rot13 all the indexes for security (but with more enterprise)",
		Introduced:     NewVersion(3, 25),
		Deprecated:     nil,
		Progress:       0,
		Created:        testTime,
		LastUpdated:    nil,
		NonDestructive: false,
		ApplyReverse:   false,
		Errors:         []MigrationError{},
	},
	{
		ID:             5,
		Team:           "codeintel",
		Component:      "lsif_data_documents",
		Description:    "denormalize counts (but with more enterprise)",
		Introduced:     NewVersion(3, 26),
		Deprecated:     newVersionPtr(3, 28),
		Progress:       0.5,
		Created:        testTime.Add(time.Hour * 1),
		LastUpdated:    timePtr(testTime.Add(time.Hour * 2)),
		NonDestructive: true,
		ApplyReverse:   false,
		Errors:         []MigrationError{},
	},
}

func timePtr(t time.Time) *time.Time { return &t }

func newVersionPtr(major, minor int) *Version {
	v := NewVersion(major, minor)
	return &v
}

func testStore(t *testing.T, db dbutil.DB) *Store {
	store := NewStoreWithDB(db)

	for i := range testMigrations {
		if err := insertMigration(store, testMigrations[i], false); err != nil {
			t.Fatalf("unexpected error inserting migration: %s", err)
		}
	}

	for i := range testEnterpriseMigrations {
		if err := insertMigration(store, testEnterpriseMigrations[i], true); err != nil {
			t.Fatalf("unexpected error inserting enterprise migration: %s", err)
		}
	}

	return store
}

func insertMigration(store *Store, migration Migration, enterpriseOnly bool) error {
	var deprecatedMajor, deprecatedMinor *int
	if migration.Deprecated != nil {
		deprecatedMajor = &migration.Deprecated.Major
		deprecatedMinor = &migration.Deprecated.Minor
	}

	query := sqlf.Sprintf(`
		INSERT INTO out_of_band_migrations (
			id,
			team,
			component,
			description,
			introduced_version_major,
			introduced_version_minor,
			deprecated_version_major,
			deprecated_version_minor,
			progress,
			created,
			last_updated,
			non_destructive,
			apply_reverse,
			is_enterprise
		) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
	`,
		migration.ID,
		migration.Team,
		migration.Component,
		migration.Description,
		migration.Introduced.Major,
		migration.Introduced.Minor,
		deprecatedMajor,
		deprecatedMinor,
		migration.Progress,
		migration.Created,
		migration.LastUpdated,
		migration.NonDestructive,
		migration.ApplyReverse,
		enterpriseOnly,
	)

	if err := store.Store.Exec(context.Background(), query); err != nil {
		return err
	}

	for _, err := range migration.Errors {
		query := sqlf.Sprintf(`
			INSERT INTO out_of_band_migrations_errors (
				migration_id,
				message,
				created
			) VALUES (%s, %s, %s)
		`,
			migration.ID,
			err.Message,
			err.Created,
		)

		if err := store.Store.Exec(context.Background(), query); err != nil {
			return err
		}
	}

	return nil
}
