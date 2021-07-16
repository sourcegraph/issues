package batch

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

// Inserter allows for bulk updates to a single Postgres table.
type Inserter struct {
	db               dbutil.DB
	numColumns       int
	maxBatchSize     int
	batch            []interface{}
	queryPrefix      string
	querySuffix      string
	queryOnConflict  string
	returningSuffix  string
	returningScanner ReturningScanner
}

type ReturningScanner func(rows *sql.Rows) error

// InsertValues creates a new batch inserter using the given database handle, table name, and
// column names, then reads from the given channel as if they specify values for a single row.
// The inserter will be flushed and any error that occurred during insertion or flush will be
// returned.
func InsertValues(ctx context.Context, db dbutil.DB, tableName string, columnNames []string, values <-chan []interface{}) error {
	return WithInserter(ctx, db, tableName, columnNames, func(inserter *Inserter) error {
	outer:
		for {
			select {
			case rowValues, ok := <-values:
				if !ok {
					break outer
				}

				if err := inserter.Insert(ctx, rowValues...); err != nil {
					return err
				}

			case <-ctx.Done():
				break outer
			}
		}

		return nil
	})
}

// WithInserter creates a new batch inserter using the given database handle, table name,
// and column names, then calls the given function with the new inserter as a parameter.
// The inserter will be flushed regardless of the error condition of the given function.
// Any error returned from the given function will be decorated with the inserter's flush
// error, if one occurs.
func WithInserter(
	ctx context.Context,
	db dbutil.DB,
	tableName string,
	columnNames []string,
	f func(inserter *Inserter) error,
) (err error) {
	inserter := NewInserter(ctx, db, tableName, columnNames...)
	return with(ctx, inserter, f)
}

// WithInserterWithReturn creates a new batch inserter using the given database handle,
// table name, column names, returning columns and returning scanner, then calls the given
// function with the new inserter as a parameter. The inserter will be flushed regardless
// of the error condition of the given function. Any error returned from the given function
// will be decorated with the inserter's flush error, if one occurs.
func WithInserterWithReturn(
	ctx context.Context,
	db dbutil.DB,
	tableName string,
	columnNames []string,
	returningColumnNames []string,
	returningScanner ReturningScanner,
	f func(inserter *Inserter) error,
) (err error) {
	inserter := NewInserterWithReturn(ctx, db, tableName, columnNames, returningColumnNames, returningScanner)
	return with(ctx, inserter, f)
}

func with(ctx context.Context, inserter *Inserter, f func(inserter *Inserter) error) (err error) {
	defer func() {
		if flushErr := inserter.Flush(ctx); flushErr != nil {
			err = multierror.Append(err, errors.Wrap(flushErr, "inserter.Flush"))
		}
	}()

	return f(inserter)
}

// NewInserter creates a new batch inserter using the given database handle, table name,
// and column names. For performance and atomicity, handle should be a transaction.
func NewInserter(ctx context.Context, db dbutil.DB, tableName string, columnNames ...string) *Inserter {
	return NewInserterWithReturn(ctx, db, tableName, columnNames, nil, nil)
}

// NewInserterWithReturn creates a new batch inserter using the given database handle, table
// name, insert column names, and column names to scan on each inserted row. The given scanner
// will be called once for each row inserted into the target table. Beware that this function
// may not be called immediately after a call to Insert as rows are only flushed once the
// current batch is full (or on explicit flush). For performance and atomicity, handle should
// be a transaction.
func NewInserterWithReturn(
	ctx context.Context,
	db dbutil.DB,
	tableName string,
	columnNames []string,
	returningColumnNames []string,
	returningScanner ReturningScanner,
) *Inserter {
	numColumns := len(columnNames)
	maxBatchSize := getMaxBatchSize(numColumns)
	queryPrefix := makeQueryPrefix(tableName, columnNames)
	querySuffix := makeQuerySuffix(numColumns)
	returningSuffix := makeReturningSuffix(returningColumnNames)

	return &Inserter{
		db:               db,
		numColumns:       numColumns,
		maxBatchSize:     maxBatchSize,
		batch:            make([]interface{}, 0, maxBatchSize),
		queryPrefix:      queryPrefix,
		querySuffix:      querySuffix,
		returningSuffix:  returningSuffix,
		returningScanner: returningScanner,
	}
}

// NewInserterWithReturn creates a new batch inserter using the given database handle, table
// name, insert column names, and column names to scan on each inserted row. The given scanner
// will be called once for each row inserted into the target table. Beware that this function
// may not be called immediately after a call to Insert as rows are only flushed once the
// current batch is full (or on explicit flush). For performance and atomicity, handle should
// be a transaction.
func NewInserterWithOnConflict(
	ctx context.Context,
	db dbutil.DB,
	tableName string,
	columnNames []string,
	onConflictAction string,
) *Inserter {
	numColumns := len(columnNames)
	maxBatchSize := getMaxBatchSize(numColumns)
	queryPrefix := makeQueryPrefix(tableName, columnNames)
	queryOnConflict := makeOnConflict(onConflictAction)
	querySuffix := makeQuerySuffix(numColumns)

	return &Inserter{
		db:              db,
		numColumns:      numColumns,
		maxBatchSize:    maxBatchSize,
		batch:           make([]interface{}, 0, maxBatchSize),
		queryPrefix:     queryPrefix,
		querySuffix:     querySuffix,
		queryOnConflict: queryOnConflict,
	}
}

// Insert submits a single row of values to be inserted on the next flush.
func (i *Inserter) Insert(ctx context.Context, values ...interface{}) error {
	if len(values) != i.numColumns {
		return errors.Errorf("expected %d values, got %d", i.numColumns, len(values))
	}

	i.batch = append(i.batch, values...)

	if len(i.batch) >= i.maxBatchSize {
		// Flush full batch
		return i.Flush(ctx)
	}

	return nil
}

// Flush ensures that all queued rows are inserted. This method must be invoked at the end
// of insertion to ensure that all records are flushed to the underlying Execable.
func (i *Inserter) Flush(ctx context.Context) (err error) {
	if batch := i.pop(); len(batch) != 0 {
		// Create a query with enough placeholders to match the current batch size. This should
		// generally be the full querySuffix string, except for the last call to Flush which
		// may be a partial batch.
		rows, err := i.db.QueryContext(dbconn.WithBulkInsertion(ctx, true), i.makeQuery(len(batch)), batch...)
		if err != nil {
			return err
		}
		defer func() { err = basestore.CloseRows(rows, err) }()

		for rows.Next() {
			if err := i.returningScanner(rows); err != nil {
				return err
			}
		}
	}

	return nil
}

// pop removes and returns as many values from the current batch that can be attached to a single
// insert statement. The returned values are the oldest values submitted to the batch (in order).
func (i *Inserter) pop() (batch []interface{}) {
	if len(i.batch) < i.maxBatchSize {
		batch, i.batch = i.batch, i.batch[:0]
		return batch
	}

	batch, i.batch = i.batch[:i.maxBatchSize], i.batch[i.maxBatchSize:]
	return batch
}

// makeQuery returns a parameterized SQL query that has the given number of values worth of
// placeholder variables. It is assumed that the number of values is non-zero and also is a
// multiple of the number of columns of the target table.
func (i *Inserter) makeQuery(numValues int) string {
	// Determine how many characters a single tuple of the query suffix occupies.
	// The tuples have the form `($xxxxx,$xxxxx,...)`, and all placeholders are
	// exactly five digits for uniformity. This counts 5 digits, `$`, and `,` for
	// each value, then un-counts the trailing comma, then counts the enveloping
	// `(` and `)`.
	sizeOfTuple := 7*i.numColumns - 1 + 2

	// Determine number of tuples being flushed
	numTuples := numValues / i.numColumns

	// Count commas separating tuples, then un-count the trailing comma
	suffixLength := numTuples*sizeOfTuple + numTuples - 1

	// Construct the query
	return i.queryPrefix + i.querySuffix[:suffixLength] + i.queryOnConflict + i.returningSuffix
}

// maxNumPostgresParameters is the maximum number of placeholder variables allowed by Postgres
// in a single insert statement.
const maxNumParameters = 32767

// getMaxBatchSize returns the number of rows that can be inserted into a single table with the
// given number of columns via a single insert statement.
func getMaxBatchSize(numColumns int) int {
	return (maxNumParameters / numColumns) * numColumns
}

// makeQueryPrefix creates the prefix of the batch insert statement (up to `VALUES `) using the
// given table and column names.
func makeQueryPrefix(tableName string, columnNames []string) string {
	quotedColumnNames := make([]string, 0, len(columnNames))
	for _, columnName := range columnNames {
		quotedColumnNames = append(quotedColumnNames, fmt.Sprintf(`"%s"`, columnName))
	}

	return fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES `, tableName, strings.Join(quotedColumnNames, ","))
}

var (
	querySuffixCache      = map[int]string{}
	querySuffixCacheMutex sync.Mutex
)

// makeQuerySuffix creates the suffix of the batch insert statement containing the placeholder
// variables, e.g. `($1,$2,$3),($4,$5,$6),...`. The number of rows will be the maximum number of
// _full_ rows that can be inserted in one insert statement.
//
// If a fewer number of rows should be inserted (due to flushing a partial batch), then the caller
// slice the appropriate nubmer of rows from the beginning of the string. The suffix constructed
// here is done so with this use case in mind (each placeholder is 5 digits), so finding the right
// substring index is efficient.
//
// This method is memoized.
func makeQuerySuffix(numColumns int) string {
	querySuffixCacheMutex.Lock()
	defer querySuffixCacheMutex.Unlock()
	if cache, ok := querySuffixCache[numColumns]; ok {
		return cache
	}

	qs := []byte{
		',', // Start with trailing comma for processing uniformity
	}
	for i := 0; i < maxNumParameters; i++ {
		if i%numColumns == 0 {
			// Replace previous `,` with `),(`
			qs[len(qs)-1] = ')'
			qs = append(qs, ',', '(')
		}
		qs = append(qs, []byte(fmt.Sprintf("$%05d", i+1))...)
		qs = append(qs, ',')
	}
	// Replace trailing `,` with `)`
	qs[len(qs)-1] = ')'

	// Chop off leading `),`
	querySuffix := string(qs[2:])
	querySuffixCache[numColumns] = querySuffix
	return querySuffix
}

func makeOnConflict(action string) string {
	if action == "" {
		return ""
	}

	return "ON CONFLICT " + action
}

// makeReturningSuffix creates a RETURNING ... clause of the batch insert statement, if any
// returning column names were supplied to the batcher inserter.
func makeReturningSuffix(columnNames []string) string {
	if len(columnNames) == 0 {
		return ""
	}

	return fmt.Sprintf("RETURNING %s", strings.Join(columnNames, ", "))
}
