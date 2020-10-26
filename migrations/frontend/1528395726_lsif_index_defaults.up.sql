BEGIN;

UPDATE lsif_indexes
SET
    indexer_args = '{lsif-go,--no-progress}'
WHERE
    indexer = 'sourcegraph/lsif-go:latest' AND
    indexer_args = '{}';

COMMIT;
