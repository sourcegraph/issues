BEGIN;

UPDATE lsif_indexes
SET
    indexer_args = '{lsif-go,--no-progress}'
WHERE
    indexer_args = '{lsif-go,--no-animation}';

COMMIT;
