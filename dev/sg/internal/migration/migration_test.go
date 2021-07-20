package migration

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/dev/sg/internal/db"
)

var defaultDB = db.DefaultDatabase

func TestSameMigrations(t *testing.T) {
	branch := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
	}

	if conflicts, _, _ := findConflictingMigrations(branch, branch); len(conflicts) > 0 {
		t.Errorf("Failed when comparing exacly the same migrations: %+v", branch)
	}
}

func TestBranchHasExtraMigration(t *testing.T) {
	trunk := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
	}

	branch := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "second",
		},
	}

	if conflicts, _, _ := findConflictingMigrations(trunk, branch); len(conflicts) > 0 {
		t.Errorf("Failed when comparing exacly the same migrations: %+v", branch)
	}
}

func TestTrunkHasExtraMigration(t *testing.T) {
	trunk := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "second",
		},
	}

	branch := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
	}

	_, missing, err := findConflictingMigrations(trunk, branch)
	if err != nil {
		t.Errorf("Should not error for missing a migration.")
	}

	if len(missing) != 1 || missing[0].Name != "second" {
		t.Errorf("Should have returned only 'second' was missing, instead +%v", missing)
	}
}

func TestTrunkHasConflictingMigration(t *testing.T) {
	trunk := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "trunk second",
		},
	}

	branch := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "branch second",
		},
	}

	conflicts, _, _ := findConflictingMigrations(trunk, branch)
	expected := []migrationConflict{
		{
			ID:    2,
			Main:  trunk[2],
			Local: branch[2],
		},
	}

	if diff := cmp.Diff(conflicts, expected); diff != "" {
		t.Errorf("Not conflicts expected: %s", diff)
	}
}

func TestGetOperationsSame(t *testing.T) {
	shared := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
	}

	operations, _ := getMigrationOperations(shared, shared, 1)

	if len(operations) > 0 {
		t.Errorf("Should not have any operations %+v", operations)
	}
}

func TestGetOperationsOneRenameNoDB(t *testing.T) {
	main := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "trunk second",
		},
	}

	local := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "branch second",
		},
	}

	// Since the database access only happens if we have to change the DB
	// we should _only_ get the fixup operation
	operations, _ := getMigrationOperations(main, local, 1)
	expected := []MigrationOperation{
		OpFileFixup{
			Conflict: migrationConflict{
				ID:    2,
				Main:  main[2],
				Local: local[2],
			},
		},
		OpDatabaseSync{},
	}

	if diff := cmp.Diff(operations, expected); diff != "" {
		t.Errorf("Expected, but got %+v", diff)
	}
}

func TestGetOperationsOneRenameWithDB(t *testing.T) {
	main := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "trunk second",
		},
	}

	local := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "branch second",
		},
	}

	operations, _ := getMigrationOperations(main, local, 2)
	expected := []MigrationOperation{
		OpTempfile{Migration: main[2]},
		OpDatabaseDown{Migration: local[2]},
		OpReverseTempfile{Migration: main[2]},
		OpFileFixup{
			Conflict: migrationConflict{
				ID:    2,
				Main:  main[2],
				Local: local[2],
			},
		},
		OpDatabaseSync{},
	}

	if diff := cmp.Diff(operations, expected); diff != "" {
		t.Errorf("Expected, but got %+v", diff)
	}
}

func TestGetOperationsTwoRenameWithDB(t *testing.T) {
	main := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "trunk second",
		},
		3: {
			ID:   3,
			Name: "trunk third",
		},
	}

	local := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "branch second: ends 4",
		},
		3: {
			ID:   3,
			Name: "branch third: ends 5",
		},
	}

	operations, _ := getMigrationOperations(main, local, 3)
	expected := []MigrationOperation{
		OpTempfile{Migration: main[2]},
		OpTempfile{Migration: main[3]},
		OpDatabaseDown{Migration: local[3]},
		OpDatabaseDown{Migration: local[2]},
		OpReverseTempfile{Migration: main[2]},
		OpReverseTempfile{Migration: main[3]},
		OpFileFixup{
			Conflict: migrationConflict{
				ID:    2,
				Main:  main[2],
				Local: local[2],
			},
		},
		OpFileFixup{
			Conflict: migrationConflict{
				ID:    3,
				Main:  main[3],
				Local: local[3],
			},
		},
		OpDatabaseSync{},
	}

	if diff := cmp.Diff(operations, expected); diff != "" {
		t.Errorf("Expected, but got %+v", diff)
	}
}

func TestGetOprerationsWithLopsidedMigrations(t *testing.T) {
	main := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "trunk second",
		},
	}

	local := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "branch second: ends 3",
		},
		3: {
			ID:   3,
			Name: "branch third: ends 4",
		},
		4: {
			ID:   4,
			Name: "branch fourth: ends 6",
		},
	}

	operations, _ := getMigrationOperations(main, local, 3)
	expected := []MigrationOperation{
		OpTempfile{Migration: main[2]},
		OpDatabaseDown{Migration: local[3]},
		OpDatabaseDown{Migration: local[2]},
		OpReverseTempfile{Migration: main[2]},
		OpFileFixup{
			Conflict: migrationConflict{
				ID:    2,
				Main:  main[2],
				Local: local[2],
			},
		},
		OpFileShift{Migration: local[3], Magnitude: 1},
		OpFileShift{Migration: local[4], Magnitude: 1},
		OpDatabaseSync{},
	}

	if diff := cmp.Diff(operations, expected); diff != "" {
		t.Errorf("Expected, but got %+v", diff)
	}
}

func TestGetOprerationsWithTwiceLopsidedMigrations(t *testing.T) {
	main := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "trunk second",
		},
		3: {
			ID:   3,
			Name: "trunk third",
		},
	}

	local := map[int]migration{
		1: {
			ID:   1,
			Name: "first",
		},
		2: {
			ID:   2,
			Name: "branch second: ends 4",
		},
		3: {
			ID:   3,
			Name: "branch third: ends 5",
		},
		4: {
			ID:   4,
			Name: "branch fourth: ends 7",
		},
	}

	operations, _ := getMigrationOperations(main, local, 4)
	expected := []MigrationOperation{
		OpTempfile{Migration: main[2]},
		OpTempfile{Migration: main[3]},
		OpDatabaseDown{Migration: local[4]},
		OpDatabaseDown{Migration: local[3]},
		OpDatabaseDown{Migration: local[2]},
		OpReverseTempfile{Migration: main[2]},
		OpReverseTempfile{Migration: main[3]},
		OpFileFixup{
			Conflict: migrationConflict{
				ID:    2,
				Main:  main[2],
				Local: local[2],
			},
		},
		OpFileFixup{
			Conflict: migrationConflict{
				ID:    3,
				Main:  main[3],
				Local: local[3],
			},
		},
		OpFileShift{Migration: local[4], Magnitude: 2},
		OpDatabaseSync{},
	}

	if diff := cmp.Diff(operations, expected); diff != "" {
		t.Errorf("Expected, but got %+v", diff)
	}
}
