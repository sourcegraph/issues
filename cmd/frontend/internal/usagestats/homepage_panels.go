package usagestats

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
)

const getHomepagePanelsQuery = `
-- source: cmd/frontend/internal/usagestats/homepage_panels.go:GetHomePagePanels
SELECT
  recentFilesPanelFileClicked / recentFilesPanelLoaded                         AS recentFilesClickedPercentage,
  recentSearchesPanelSearchClicked / recentSearchesPanelLoaded                 AS recentSearchClickedPercentage,
  repositoriesPanelRepoFilterClicked / repositoriesPanelLoaded                 AS recentRepositoriesClickedPercentage,
  savedSearchesPanelSearchClicked / savedSearchesPanelLoaded                   AS savedSearchesClickedPercentage,
  savedSearchesPanelCreateButtonClicked / savedSearchesPanelLoaded             AS newSavedSearchesClickedPercentage,
  recentSearchesPanelLoaded                                                    AS totalPanelViews,
  uniqueRecentFilesPanelFileClicked / uniqueRecentFilesPanelLoaded             AS usersFilesClickedPercentage,
  uniqueRecentSearchesPanelSearchClicked / uniqueRecentSearchesPanelLoaded     AS usersSearchClickedPercentage,
  uniqueRepositoriesPanelRepoFilterClicked / uniqueRepositoriesPanelLoaded     AS usersRepositoriesClickedPercentage,
  uniqueSavedSearchesPanelSearchClicked / uniqueSavedSearchesPanelLoaded       AS usersSavedSearchesClickedPercentage,
  uniqueSavedSearchesPanelCreateButtonClicked / uniqueSavedSearchesPanelLoaded AS usersNewSavedSearchesClickedPercentage,
  uniqueRecentSearchesPanelLoaded                                              AS percentUsersShown
FROM (
  SELECT
    NULLIF(COUNT(*) FILTER (WHERE name = 'RecentFilesPanelFileClicked'), 0)           :: FLOAT AS recentFilesPanelFileClicked,
    NULLIF(COUNT(*) FILTER (WHERE name = 'RecentFilesPanelLoaded'), 0)                :: FLOAT AS recentFilesPanelLoaded,
    NULLIF(COUNT(*) FILTER (WHERE name = 'RecentSearchesPanelSearchClicked'), 0)      :: FLOAT AS recentSearchesPanelSearchClicked,
    NULLIF(COUNT(*) FILTER (WHERE name = 'RecentSearchesPanelLoaded'), 0)             :: FLOAT AS recentSearchesPanelLoaded,
    NULLIF(COUNT(*) FILTER (WHERE name = 'RepositoriesPanelRepoFilterClicked'), 0)    :: FLOAT AS repositoriesPanelRepoFilterClicked,
    NULLIF(COUNT(*) FILTER (WHERE name = 'RepositoriesPanelLoaded'), 0)               :: FLOAT AS repositoriesPanelLoaded,
    NULLIF(COUNT(*) FILTER (WHERE name = 'SavedSearchesPanelSearchClicked'), 0)       :: FLOAT AS savedSearchesPanelSearchClicked,
    NULLIF(COUNT(*) FILTER (WHERE name = 'SavedSearchesPanelLoaded'), 0)              :: FLOAT AS savedSearchesPanelLoaded,
    NULLIF(COUNT(*) FILTER (WHERE name = 'SavedSearchesPanelCreateButtonClicked'), 0) :: FLOAT AS savedSearchesPanelCreateButtonClicked,

    NULLIF(COUNT(DISTINCT user_id) FILTER (WHERE name = 'RecentFilesPanelLoaded'), 0)                :: FLOAT AS uniqueRecentFilesPanelLoaded,
    NULLIF(COUNT(DISTINCT user_id) FILTER (WHERE name = 'RecentSearchesPanelSearchClicked'), 0)      :: FLOAT AS uniqueRecentSearchesPanelSearchClicked,
    NULLIF(COUNT(DISTINCT user_id) FILTER (WHERE name = 'RecentSearchesPanelLoaded'), 0)             :: FLOAT AS uniqueRecentSearchesPanelLoaded,
    NULLIF(COUNT(DISTINCT user_id) FILTER (WHERE name = 'RepositoriesPanelRepoFilterClicked'), 0)    :: FLOAT AS uniqueRepositoriesPanelRepoFilterClicked,
    NULLIF(COUNT(DISTINCT user_id) FILTER (WHERE name = 'RepositoriesPanelLoaded'), 0)               :: FLOAT AS uniqueRepositoriesPanelLoaded,
    NULLIF(COUNT(DISTINCT user_id) FILTER (WHERE name = 'SavedSearchesPanelSearchClicked'), 0)       :: FLOAT AS uniqueSavedSearchesPanelSearchClicked,
    NULLIF(COUNT(DISTINCT user_id) FILTER (WHERE name = 'SavedSearchesPanelLoaded'), 0)              :: FLOAT AS uniqueSavedSearchesPanelLoaded,
    NULLIF(COUNT(DISTINCT user_id) FILTER (WHERE name = 'SavedSearchesPanelCreateButtonClicked'), 0) :: FLOAT AS uniqueSavedSearchesPanelCreateButtonClicked
  FROM
    event_logs
  WHERE name IN (
   'RecentFilesPanelFileClicked',
   'RecentFilesPanelLoaded',
   'RecentSearchesPanelSearchClicked',
   'RecentSearchesPanelLoaded',
   'RepositoriesPanelRepoFilterClicked',
   'RepositoriesPanelLoaded',
   'SavedSearchesPanelSearchClicked',
   'SavedSearchesPanelLoaded',
   'SavedSearchesPanelCreateButtonClicked',
   'RecentFilesPanelFileClicked',
   'RecentFilesPanelLoaded'
  ) AND DATE(TIMEZONE('UTC', timestamp)) >= DATE_TRUNC('week', current_date)
) sub
`

func GetHomepagePanels(ctx context.Context) (*types.HomepagePanels, error) {
	var p types.HomepagePanels
	return &p, dbconn.Global.QueryRowContext(ctx, getHomepagePanelsQuery).Scan(
		&p.RecentFilesClickedPercentage,
		&p.RecentSearchClickedPercentage,
		&p.RecentRepositoriesClickedPercentage,
		&p.SavedSearchesClickedPercentage,
		&p.NewSavedSearchesClickedPercentage,
		&p.TotalPanelViews,
		&p.UsersFilesClickedPercentage,
		&p.UsersSearchClickedPercentage,
		&p.UsersRepositoriesClickedPercentage,
		&p.UsersSavedSearchesClickedPercentage,
		&p.UsersNewSavedSearchesClickedPercentage,
		&p.PercentUsersShown,
	)
}
