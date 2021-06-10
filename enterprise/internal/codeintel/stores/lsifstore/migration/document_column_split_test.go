package migration

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/lsifstore"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/semantic"
)

func TestDocumentColumnSplitMigrator(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	store := lsifstore.NewStore(db, &observation.TestContext)
	migrator := NewDocumentColumnSplitMigrator(store, 250)
	serializer := lsifstore.NewSerializer()

	assertProgress := func(expectedProgress float64) {
		if progress, err := migrator.Progress(context.Background()); err != nil {
			t.Fatalf("unexpected error querying progress: %s", err)
		} else if progress != expectedProgress {
			t.Errorf("unexpected progress. want=%.2f have=%.2f", expectedProgress, progress)
		}
	}

	scanHoverCounts := func(rows *sql.Rows, queryErr error) (counts []int, err error) {
		if queryErr != nil {
			return nil, queryErr
		}
		defer func() { err = basestore.CloseRows(rows, err) }()

		for rows.Next() {
			var rawData []byte
			if err := rows.Scan(&rawData); err != nil {
				return nil, err
			}

			encoded := lsifstore.MarshalledDocumentData{
				HoverResults: rawData,
			}
			decoded, err := serializer.UnmarshalDocumentData(encoded)
			if err != nil {
				return nil, err
			}

			counts = append(counts, len(decoded.HoverResults))
		}

		return counts, nil
	}

	assertCounts := func(expectedCounts []int) {
		query := sqlf.Sprintf(`SELECT hovers FROM lsif_data_documents ORDER BY path`)

		if counts, err := scanHoverCounts(store.Query(context.Background(), query)); err != nil {
			t.Fatalf("unexpected error querying num hovers: %s", err)
		} else if diff := cmp.Diff(expectedCounts, counts); diff != "" {
			t.Errorf("unexpected counts (-want +got):\n%s", diff)
		}
	}

	n := 500
	expectedCounts := make([]int, 0, n)
	hovers := make(map[semantic.ID]string, n)
	diagnostics := make([]semantic.DiagnosticData, 0, n)

	for i := 0; i < n; i++ {
		expectedCounts = append(expectedCounts, i+1)
		hovers[semantic.ID(strconv.Itoa(i))] = fmt.Sprintf("h%d", i)
		diagnostics = append(diagnostics, semantic.DiagnosticData{Code: fmt.Sprintf("c%d", i)})

		data, err := serializer.MarshalLegacyDocumentData(semantic.DocumentData{
			HoverResults: hovers,
			Diagnostics:  diagnostics,
		})
		if err != nil {
			t.Fatalf("unexpected error serializing document data: %s", err)
		}

		if err := store.Exec(context.Background(), sqlf.Sprintf(
			"INSERT INTO lsif_data_documents (dump_id, path, data, schema_version, num_diagnostics) VALUES (%s, %s, %s, 2, %s)",
			42+i/(n/2), // 50% id=42, 50% id=43
			fmt.Sprintf("p%04d", i),
			data,
			len(diagnostics),
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
