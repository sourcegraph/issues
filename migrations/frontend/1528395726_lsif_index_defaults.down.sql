BEGIN;

UPDATE lsif_indexes
SET
    indexer_args = '{}'
WHERE
    indexer = 'sourcegraph/lsif-go:latest' AND
    indexer_args = '{lsif-go,--no-progress}';

COMMIT;
