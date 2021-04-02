package migration

import (
	"context"
	"sort"

	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/lsifstore"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/batch"
	"github.com/sourcegraph/sourcegraph/internal/oobmigration"
)

// Migrator is a code-intelligence-specific out-of-band migration runner. This migrator can
// be configured by supplying a different driver instance, which controls the update behavior
// over every matching row in the migration set.
//
// Code intelligence tables are very large and using a full table scan count is too expensvie
// to use in an out-of-band migration. For each table we need to perform a migration over, we
// introduce a second aggregate table that tracks the minimum and maximum schema version of
// each data record  associated with a particular upload record.
//
// We have the following assumptions about the schema (for a configured table T):
//
// 1. There is an index on T.dump_id
// 2. For each distinct dump_id in table T, there is a corresponding row in table
//    T_schema_version. This invariant is kept up to date via triggers on insert.
// 3. Table T_schema_version has the following schema:
//
//    CREATE TABLE T_schema_versions (
//        dump_id            integer PRIMARY KEY NOT NULL,
//        min_schema_version integer,
//        max_schema_version integer
//    );
//
// When selecting a set of candidate records to migrate, we first use the each upload record's
// schema version bounds to determine if there are still records associated with that upload
// that may still need migrating. This set allows us to use the dump_id index on the target
// table. These counts can be maintained efficiently within the same transaction as a batch
// of migration updates. This requires counting within a small indexed subset of the original
// table. When checking progress, we can efficiently do a full-table on the much smaller
// aggregate table.
type Migrator struct {
	store   *lsifstore.Store
	driver  migrationDriver
	options migratorOptions
}

type migratorOptions struct {
	// tableName is the name of the table undergoing migration.
	tableName string

	// targetVersion is the value of the row's schema version after an up migration.
	targetVersion int

	// batchSize limits the number of rows that will be scanned on each call to Up/Down.
	batchSize int

	// fields is an ordered set of fields used to construct temporary tables and update queries.
	fields []fieldSpec
}

type fieldSpec struct {
	// name is the name of the column.
	name string

	// postgresType is the type (with modifiers) of the column.
	postgresType string

	// primaryKey indicates that the field is part of a composite primary key.
	primaryKey bool

	// readOnly indicates that the field should be skipped on updates.
	readOnly bool

	// updateOnly indicates that the field should be skipped on selects.
	updateOnly bool
}

type updateSpec struct {
	// dumpID is the identifier of the associated upload record.
	dumpID int

	// fieldValues indicates the values that should be written back to the table. This must have
	// the same number of values as the set of primary keys plus any additional non-selectOnly
	// fields supplied via the migrator's fields option.
	fieldValues []interface{}
}

type migrationDriver interface {
	// MigrateRowUp determines which fields to update for the given row. The scanner will receive
	// the values of the primary keys plus any additional non-updateOnly fields supplied via the
	// migrator's fields option.
	MigrateRowUp(scanner scanner) (updateSpec, error)

	// MigrateRowDown undoes the migration for the given row.  The scanner will receive the values
	// of the primary keys plus any additional non-updateOnly fields supplied via the migrator's
	// fields option.
	MigrateRowDown(scanner scanner) (updateSpec, error)
}

// driverFunc is the type of MigrateRowUp and MigrateRowDown.
type driverFunc func(scanner scanner) (updateSpec, error)

type scanner interface {
	Scan(dest ...interface{}) error
}

func newMigrator(store *lsifstore.Store, driver migrationDriver, options migratorOptions) oobmigration.Migrator {
	return &Migrator{
		store:   store,
		driver:  driver,
		options: options,
	}
}

// Progress returns the ratio between the number of upload records that have been completely
// migrated over the total number of upload records. A record is migrated if its schema version
// is no less than the target migration version.
func (m *Migrator) Progress(ctx context.Context) (float64, error) {
	progress, _, err := basestore.ScanFirstFloat(m.store.Query(ctx, sqlf.Sprintf(
		migratorProgressQuery,
		sqlf.Sprintf(m.options.tableName),
		m.options.targetVersion,
		sqlf.Sprintf(m.options.tableName),
	)))
	if err != nil {
		return 0, err
	}

	return progress, nil
}

const migratorProgressQuery = `
-- source: enterprise/internal/codeintel/stores/lsifstore/migration/migrator.go:Progress
SELECT CASE c2.count WHEN 0 THEN 1 ELSE cast(c1.count as float) / cast(c2.count as float) END FROM
	(SELECT COUNT(*) as count FROM %s_schema_versions WHERE min_schema_version >= %s) c1,
	(SELECT COUNT(*) as count FROM %s_schema_versions) c2
`

// Up runs a batch of the migration.
func (m *Migrator) Up(ctx context.Context) (err error) {
	return m.run(ctx, m.options.targetVersion-1, m.options.targetVersion, m.driver.MigrateRowUp)
}

// Down runs a batch of the migration in reverse.
func (m *Migrator) Down(ctx context.Context) error {
	return m.run(ctx, m.options.targetVersion, m.options.targetVersion-1, m.driver.MigrateRowDown)
}

// run performs a batch of updates with the given driver function. Records with the given source version
// will be selected for candidacy, and their version will match the given target version after an update.
func (m *Migrator) run(ctx context.Context, sourceVersion, targetVersion int, driverFunc driverFunc) (err error) {
	tx, err := m.store.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() { err = tx.Done(err) }()

	// Perform the actual batch of migrations
	ids, err := m.selectAndUpdate(ctx, tx, sourceVersion, targetVersion, driverFunc)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	idQueries := make([]*sqlf.Query, 0, len(ids))
	for _, id := range ids {
		idQueries = append(idQueries, sqlf.Sprintf("%s", id))
	}

	return tx.Exec(ctx, sqlf.Sprintf(runQuery, sqlf.Sprintf(m.options.tableName), sqlf.Sprintf(m.options.tableName), sqlf.Join(idQueries, ", ")))
}

const runQuery = `
-- source: enterprise/internal/codeintel/stores/lsifstore/migration/migrator.go:run
INSERT INTO %s_schema_versions
SELECT
	dump_id,
	MIN(schema_version) as min_schema_version,
	MAX(schema_version) as max_schema_version
FROM
	%s
WHERE
	dump_id IN (%s)
GROUP BY
	dump_id
ON CONFLICT (dump_id) DO UPDATE SET
	min_schema_version = EXCLUDED.min_schema_version,
	max_schema_version = EXCLUDED.max_schema_version
`

// selectAndUpdate selects a batch of records from the configured table with the given source
// version, then performs an update on each matching row with the new values given from an
// invocation of the given driver function. This method returns a deduplicated and ordered set
// of upload identifiers denoting the complete set of uploads whose records were modified by
// this batch of updates.
func (m *Migrator) selectAndUpdate(ctx context.Context, tx *lsifstore.Store, sourceVersion, targetVersion int, driverFunc driverFunc) ([]int, error) {
	// Note: we can't pipeline this as you can't have an open rows object and
	// execute another unrelated query using the same database handle.
	updateSpecs, err := m.selectAndProcess(ctx, tx, sourceVersion, driverFunc)
	if err != nil {
		return nil, err
	}

	// TODO - document, rename
	if err := something(ctx, tx, m.options.fields, updateSpecs); err != nil {
		return nil, err
	}

	// TODO - document, rename
	if err := something2(ctx, tx, m.options.tableName, targetVersion, m.options.fields); err != nil {
		return nil, err
	}

	idMap := map[int]struct{}{}
	for _, spec := range updateSpecs {
		idMap[spec.dumpID] = struct{}{}
	}

	ids := make([]int, 0, len(idMap))
	for id := range idMap {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	return ids, nil
}

// selectAndProcess selects a batch of records from the configured table with the given version and
// returns the update specifications after running the given driver function on each matching row.
// The records selected by this method are locked (via select for update) in the given transaction.
func (m *Migrator) selectAndProcess(ctx context.Context, tx *lsifstore.Store, version int, driverFunc driverFunc) ([]updateSpec, error) {
	fieldQueries := make([]*sqlf.Query, 0, len(m.options.fields))
	for _, field := range m.options.fields {
		if !field.updateOnly {
			fieldQueries = append(fieldQueries, sqlf.Sprintf(field.name))
		}
	}

	rows, err := tx.Query(ctx, sqlf.Sprintf(
		selectAndProcessQuery,
		sqlf.Sprintf(m.options.tableName),
		version,
		version,
		sqlf.Join(fieldQueries, ", "),
		sqlf.Sprintf(m.options.tableName),
		version,
		m.options.batchSize,
	))
	if err != nil {
		return nil, err
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	updateSpecs := make([]updateSpec, 0, m.options.batchSize)
	for rows.Next() {
		spec, err := driverFunc(rows)
		if err != nil {
			return nil, err
		}

		updateSpecs = append(updateSpecs, spec)
	}

	return updateSpecs, nil
}

const selectAndProcessQuery = `
-- source: enterprise/internal/codeintel/stores/lsifstore/migration/migrator.go:selectAndProcess
WITH candidates AS (
	SELECT dump_id
	FROM %s_schema_versions
	WHERE
		min_schema_version <= %s AND
		max_schema_version >= %s
)
SELECT %s
FROM %s
WHERE
	dump_id IN (SELECT dump_id FROM candidates) AND
	schema_version = %s
ORDER BY dump_id
LIMIT %s
FOR UPDATE SKIP LOCKED
`

// TODO - extract
// TODO - rename
// TODO - document
// TODO - re-parameterize
func something(ctx context.Context, lsifStore *lsifstore.Store, fieldSpecs []fieldSpec, updateSpecs []updateSpec) error {
	names := make([]string, 0, len(fieldSpecs))
	namesAndTypes := make([]*sqlf.Query, 0, len(fieldSpecs))

	for _, field := range fieldSpecs {
		if field.readOnly {
			continue
		}

		names = append(names, field.name)
		namesAndTypes = append(namesAndTypes, sqlf.Sprintf(field.name+" "+field.postgresType))
	}

	if err := lsifStore.Exec(ctx, sqlf.Sprintf(somethingQuery, sqlf.Join(namesAndTypes, ", "))); err != nil {
		return err
	}

	inserter := batch.NewBatchInserter(ctx, lsifStore.Handle().DB(), "t_target", names...)

	for _, spec := range updateSpecs {
		if err := inserter.Insert(ctx, spec.fieldValues...); err != nil {
			return err
		}
	}
	if err := inserter.Flush(ctx); err != nil {
		return err
	}

	return nil
}

// TODO - rename
const somethingQuery = `
-- source: enterprise/internal/codeintel/stores/lsifstore/migration/migrator.go:something
CREATE TEMPORARY TABLE t_target (%s) ON COMMIT DROP
`

// TODO - extract
// TODO - rename
// TODO - document
// TODO - re-parameterize
func something2(ctx context.Context, tx *lsifstore.Store, tableName string, targetVersion int, fieldSpecs []fieldSpec) error {
	conditions := make([]*sqlf.Query, 0, len(fieldSpecs))
	assignments := make([]*sqlf.Query, 0, len(fieldSpecs))

	for _, field := range fieldSpecs {
		sub := field.name + "= src." + field.name

		if field.primaryKey {
			// disambiguate fields
			conditions = append(conditions, sqlf.Sprintf("dest."+sub))
		} else if !field.readOnly {
			assignments = append(assignments, sqlf.Sprintf(sub))
		}
	}

	return tx.Exec(ctx, sqlf.Sprintf(
		something2Query,
		sqlf.Sprintf(tableName),
		sqlf.Join(assignments, ", "),
		targetVersion,
		sqlf.Join(conditions, " AND "),
	))
}

// TODO - rename
const something2Query = `
-- source: enterprise/internal/codeintel/stores/lsifstore/migration/migrator.go:something2
UPDATE %s dest
SET %s, schema_version = %s
FROM t_target src
WHERE %s
`
