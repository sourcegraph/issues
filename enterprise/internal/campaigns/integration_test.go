package campaigns

import (
	"database/sql"
	"flag"
	"strings"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/db/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

var dsn = flag.String("dsn", "", "Database connection string to use in integration tests")

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	dbtesting.SetupGlobalTestDB(t)
	db := dbtest.NewDB(t, *dsn)

	userID := insertTestUser(t, db)

	t.Run("Store", func(t *testing.T) {
		t.Run("Campaigns", storeTest(db, testStoreCampaigns))
		t.Run("Changesets", storeTest(db, testStoreChangesets))
		t.Run("ChangesetEvents", storeTest(db, testStoreChangesetEvents))
		t.Run("ListChangesetSyncData", storeTest(db, testStoreListChangesetSyncData))
		t.Run("CampaignSpecs", storeTest(db, testStoreCampaignSpecs))
		t.Run("ChangesetSpecs", storeTest(db, testStoreChangesetSpecs))
	})

	t.Run("GitHubWebhook", testGitHubWebhook(db, userID))
	t.Run("BitbucketWebhook", testBitbucketWebhook(db, userID))
	t.Run("GitLabWebhook", testGitLabWebhook(db, userID))
}

func truncateTables(t *testing.T, db *sql.DB, tables ...string) {
	t.Helper()

	_, err := db.Exec("TRUNCATE " + strings.Join(tables, ", ") + " RESTART IDENTITY")
	if err != nil {
		t.Fatal(err)
	}
}

func insertTestOrg(t *testing.T, db *sql.DB) (orgID int32) {
	t.Helper()

	err := db.QueryRow("INSERT INTO orgs (name) VALUES ('bbs-org') RETURNING id").Scan(&orgID)
	if err != nil {
		t.Fatal(err)
	}

	return orgID
}

func insertTestUser(t *testing.T, db *sql.DB) (userID int32) {
	t.Helper()

	err := db.QueryRow("INSERT INTO users (username) VALUES ('bbs-admin') RETURNING id").Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}

	return userID
}
