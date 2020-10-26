package batch

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

// BatchInserter allows for bulk updates to a single Postgres table.
type BatchInserter struct {
	db           dbutil.DB
	numColumns   int
	maxBatchSize int
	batch        []interface{}
	queryPrefix  string
	querySuffix  string
}

// NewBatchInserter creates a new batch inserter using the given database handle,
// table name, and column names. For performance and atomicity, handle should be
// a transaction.
func NewBatchInserter(ctx context.Context, db dbutil.DB, tableName string, columnNames ...string) *BatchInserter {
	numColumns := len(columnNames)
	maxBatchSize := getMaxBatchSize(numColumns)
	queryPrefix := makeQueryPrefix(tableName, columnNames)
	querySuffix := makeQuerySuffix(numColumns)

	return &BatchInserter{
		db:           db,
		numColumns:   numColumns,
		maxBatchSize: maxBatchSize,
		batch:        make([]interface{}, 0, maxBatchSize),
		queryPrefix:  queryPrefix,
		querySuffix:  querySuffix,
	}
}

// Insert submits a single row of values to be inserted on the next flush.
func (i *BatchInserter) Insert(ctx context.Context, values ...interface{}) error {
	if len(values) != i.numColumns {
		return fmt.Errorf("expected %d values, got %d", i.numColumns, len(values))
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
func (i *BatchInserter) Flush(ctx context.Context) error {
	if batch := i.pop(); len(batch) != 0 {
		// Create a query with enough placeholders to match the current batch size. This should
		// generally be the full querySuffix string, except for the last call to Flush which
		// may be a partial batch.
		if _, err := i.db.ExecContext(dbconn.WithBulkInsertion(ctx, true), i.makeQuery(len(batch)), batch...); err != nil {
			return err
		}
	}

	return nil
}

// pop removes and returns as many values from the current batch that can be attached to a single
// insert statement. The returned values are the oldest values submitted to the batch (in order).
func (i *BatchInserter) pop() (batch []interface{}) {
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
func (i *BatchInserter) makeQuery(numValues int) string {
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
	return i.queryPrefix + i.querySuffix[:suffixLength]
}

// maxNumPostgresParameters is the maximum number of placeholder variables allowed by Postgres
// in a single insert statement.
const maxNumParameters = 32767

// getMaxBatchSize returns the number of rows that can be inserted into a single table with the
// give number of columns via a single insert statement.
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

var querySuffixCache = map[int]string{}
var querySuffixCacheMutex sync.Mutex

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
