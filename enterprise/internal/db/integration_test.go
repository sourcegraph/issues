package db

import (
	"flag"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/db/dbtest"
)

var dsn = flag.String("dsn", "", "Database connection string to use in integration tests")

func TestIntegration_PermsStore(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	db := dbtest.NewDB(t, *dsn)

	for _, tc := range []struct {
		name string
		test func(*testing.T)
	}{
		{"LoadUserPermissions", testPermsStore_LoadUserPermissions(db)},
		{"LoadRepoPermissions", testPermsStore_LoadRepoPermissions(db)},
		{"SetUserPermissions", testPermsStore_SetUserPermissions(db)},
		{"SetRepoPermissions", testPermsStore_SetRepoPermissions(db)},
		{"TouchRepoPermissions", testPermsStore_TouchRepoPermissions(db)},
		{"LoadUserPendingPermissions", testPermsStore_LoadUserPendingPermissions(db)},
		{"SetRepoPendingPermissions", testPermsStore_SetRepoPendingPermissions(db)},
		{"ListPendingUsers", testPermsStore_ListPendingUsers(db)},
		{"GrantPendingPermissions", testPermsStore_GrantPendingPermissions(db)},
		{"SetPendingPermissionsAfterGrant", testPermsStore_SetPendingPermissionsAfterGrant(db)},
		{"DeleteAllUserPermissions", testPermsStore_DeleteAllUserPermissions(db)},
		{"DeleteAllUserPendingPermissions", testPermsStore_DeleteAllUserPendingPermissions(db)},
		{"DatabaseDeadlocks", testPermsStore_DatabaseDeadlocks(db)},

		{"ListExternalAccounts", testPermsStore_ListExternalAccounts(db)},
		{"GetUserIDsByExternalAccounts", testPermsStore_GetUserIDsByExternalAccounts(db)},

		{"UserIDsWithNoPerms", testPermsStore_UserIDsWithNoPerms(db)},
		{"RepoIDsWithNoPerms", testPermsStore_RepoIDsWithNoPerms(db)},
		{"UserIDsWithOldestPerms", testPermsStore_UserIDsWithOldestPerms(db)},
		{"ReposIDsWithOldestPerms", testPermsStore_ReposIDsWithOldestPerms(db)},
		{"Metrics", testPermsStore_Metrics(db)},
	} {
		t.Run(tc.name, tc.test)
	}
}
