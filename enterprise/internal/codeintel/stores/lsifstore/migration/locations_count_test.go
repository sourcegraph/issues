package migration

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/lsifstore"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/semantic"
)

func TestLocationsCountMigrator(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	store := lsifstore.NewStore(db, &observation.TestContext)
	migrator := NewLocationsCountMigrator(store, "lsif_data_definitions", 250)
	serializer := lsifstore.NewSerializer()

	assertProgress := func(expectedProgress float64) {
		if progress, err := migrator.Progress(context.Background()); err != nil {
			t.Fatalf("unexpected error querying progress: %s", err)
		} else if progress != expectedProgress {
			t.Errorf("unexpected progress. want=%.2f have=%.2f", expectedProgress, progress)
		}
	}

	assertCounts := func(expectedCounts []int) {
		query := sqlf.Sprintf(`SELECT num_locations FROM lsif_data_definitions ORDER BY scheme, identifier`)

		if counts, err := basestore.ScanInts(store.Query(context.Background(), query)); err != nil {
			t.Fatalf("unexpected error querying num diagnostics: %s", err)
		} else if diff := cmp.Diff(expectedCounts, counts); diff != "" {
			t.Errorf("unexpected counts (-want +got):\n%s", diff)
		}
	}

	n := 500
	expectedCounts := make([]int, 0, n)
	locations := make([]semantic.LocationData, 0, n)

	for i := 0; i < n; i++ {
		expectedCounts = append(expectedCounts, i+1)
		locations = append(locations, semantic.LocationData{URI: fmt.Sprintf("file://%d", i)})

		data, err := serializer.MarshalLocations(locations)
		if err != nil {
			t.Fatalf("unexpected error serializing locations: %s", err)
		}

		if err := store.Exec(context.Background(), sqlf.Sprintf(
			"INSERT INTO lsif_data_definitions (dump_id, scheme, identifier, data, schema_version, num_locations) VALUES (%s, %s, %s, %s, 1, 0)",
			42+i/(n/2), // 50% id=42, 50% id=43
			fmt.Sprintf("s%04d", i),
			fmt.Sprintf("i%04d", i),
			data,
		)); err != nil {
			t.Fatalf("unexpected error inserting row: %s", err)
		}
	}

	assertProgress(0)

	if err := migrator.Up(context.Background()); err != nil {
		t.Fatalf("unexpected error performing up migration: %s", err)
	}
	assertProgress(0.5)

	if err := migrator.Up(context.Background()); err != nil {
		t.Fatalf("unexpected error performing up migration: %s", err)
	}
	assertProgress(1)

	assertCounts(expectedCounts)

	if err := migrator.Down(context.Background()); err != nil {
		t.Fatalf("unexpected error performing down migration: %s", err)
	}
	assertProgress(0.5)

	if err := migrator.Down(context.Background()); err != nil {
		t.Fatalf("unexpected error performing down migration: %s", err)
	}
	assertProgress(0)
}
