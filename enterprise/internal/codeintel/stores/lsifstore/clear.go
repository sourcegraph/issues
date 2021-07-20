package lsifstore

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/keegancsmith/sqlf"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/internal/observation"
)

var tableNames = []string{
	"lsif_data_metadata",
	"lsif_data_documents",
	"lsif_data_documents_schema_versions",
	"lsif_data_result_chunks",
	"lsif_data_definitions",
	"lsif_data_definitions_schema_versions",
	"lsif_data_references",
	"lsif_data_references_schema_versions",
}

func (s *Store) Clear(ctx context.Context, bundleIDs ...int) (err error) {
	ctx, traceLog, endObservation := s.operations.clear.WithAndLogger(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("numBundleIDs", len(bundleIDs)),
		log.String("bundleIDs", intsToString(bundleIDs)),
	}})
	defer endObservation(1, observation.Args{})

	if len(bundleIDs) == 0 {
		return nil
	}

	// Ensure ids are sorted so that we take row locks during the
	// DELETE query in a determinstic order. This should prevent
	// deadlocks with other queries that mass update the same table.
	sort.Ints(bundleIDs)

	var ids []*sqlf.Query
	for _, bundleID := range bundleIDs {
		ids = append(ids, sqlf.Sprintf("%d", bundleID))
	}

	tx, err := s.Store.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = tx.Done(err)
	}()

	for _, tableName := range tableNames {
		traceLog(log.String("tableName", tableName))

		if err := tx.Exec(ctx, sqlf.Sprintf(clearQuery, sqlf.Sprintf(tableName), sqlf.Join(ids, ","))); err != nil {
			return err
		}
	}

	return nil
}

const clearQuery = `
-- source: enterprise/internal/codeintel/stores/lsifstore/clear.go:Clear
DELETE FROM %s WHERE dump_id IN (%s)
`

func intsToString(vs []int) string {
	strs := make([]string, 0, len(vs))
	for _, v := range vs {
		strs = append(strs, strconv.Itoa(v))
	}

	return strings.Join(strs, ", ")
}
