BEGIN;

UPDATE lsif_indexes
SET
    indexer_args = '{lsif-go,--no-animation}'
WHERE
    indexer_args = '{lsif-go,--no-progress}';

COMMIT;
