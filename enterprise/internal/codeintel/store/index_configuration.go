package store

import (
	"context"
	"database/sql"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
)

// IndexConfiguration stores the index configuration for a repository.
type IndexConfiguration struct {
	ID           int    `json:"id"`
	RepositoryID int    `json:"repository_id"`
	Data         []byte `json:"data"`
}

// scanIndexConfigurations scans a slice of index configurations from the return value of `*store.query`.
func scanIndexConfigurations(rows *sql.Rows, queryErr error) (_ []IndexConfiguration, err error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	var indexConfigurations []IndexConfiguration
	for rows.Next() {
		var indexConfiguration IndexConfiguration
		if err := rows.Scan(
			&indexConfiguration.ID,
			&indexConfiguration.RepositoryID,
			&indexConfiguration.Data,
		); err != nil {
			return nil, err
		}

		indexConfigurations = append(indexConfigurations, indexConfiguration)
	}

	return indexConfigurations, nil
}

// scanFirstIndexConfiguration scans a slice of index configurations from the return value of `*store.query`
// and returns the first.
func scanFirstIndexConfiguration(rows *sql.Rows, err error) (IndexConfiguration, bool, error) {
	indexConfigurations, err := scanIndexConfigurations(rows, err)
	if err != nil || len(indexConfigurations) == 0 {
		return IndexConfiguration{}, false, err
	}
	return indexConfigurations[0], true, nil
}

// GetRepositoriesWithIndexConfiguration returns the ids of repositories explicit index configuration.
func (s *store) GetRepositoriesWithIndexConfiguration(ctx context.Context) ([]int, error) {
	return basestore.ScanInts(s.Store.Query(ctx, sqlf.Sprintf(`SELECT c.repository_id FROM lsif_index_configuration c`)))
}

// GetIndexConfigurationByRepositoryID returns the index configuration for a repository.
func (s *store) GetIndexConfigurationByRepositoryID(ctx context.Context, repositoryID int) (IndexConfiguration, bool, error) {
	return scanFirstIndexConfiguration(s.Store.Query(ctx, sqlf.Sprintf(`
		SELECT
			c.id,
			c.repository_id,
			c.data
		FROM lsif_index_configuration c WHERE c.repository_id = %s
	`, repositoryID)))
}
