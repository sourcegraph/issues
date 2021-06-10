package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codemonitors"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

const (
	testQuery       = "repo:github\\.com/sourcegraph/sourcegraph func type:diff patternType:literal"
	testDescription = "test description"
)

type TestStore struct {
	*codemonitors.Store
}

func (s *TestStore) InsertTestMonitor(ctx context.Context, t *testing.T) (*codemonitors.Monitor, error) {
	t.Helper()

	owner := relay.MarshalID("User", actor.FromContext(ctx).UID)
	args := &graphqlbackend.CreateCodeMonitorArgs{
		Monitor: &graphqlbackend.CreateMonitorArgs{
			Namespace:   owner,
			Description: testDescription,
			Enabled:     true,
		},
		Trigger: &graphqlbackend.CreateTriggerArgs{
			Query: testQuery,
		},
		Actions: []*graphqlbackend.CreateActionArgs{
			{
				Email: &graphqlbackend.CreateActionEmailArgs{
					Enabled:    true,
					Priority:   "NORMAL",
					Recipients: []graphql.ID{owner},
					Header:     "test header 1"},
			},
			{
				Email: &graphqlbackend.CreateActionEmailArgs{
					Enabled:    true,
					Priority:   "CRITICAL",
					Recipients: []graphql.ID{owner},
					Header:     "test header 2"},
			},
		},
	}
	return s.CreateCodeMonitor(ctx, args)
}

func NewTestStoreWithStore(t *testing.T, store *codemonitors.Store) (context.Context, *TestStore) {
	ctx := backend.WithAuthzBypass(context.Background())
	now := time.Now().Truncate(time.Microsecond)
	return ctx, &TestStore{codemonitors.NewStoreWithClock(store.Handle().DB(), func() time.Time { return now })}
}

func NewTestUser(ctx context.Context, t *testing.T, db dbutil.DB) (name string, id int32, namespace graphql.ID, userContext context.Context) {
	t.Helper()

	name = "cm-user1"
	id = insertTestUser(t, db, name, true)
	namespace = relay.MarshalID("User", id)
	ctx = actor.WithActor(ctx, actor.FromUser(id))
	return name, id, namespace, ctx
}

func insertTestUser(t *testing.T, db dbutil.DB, name string, isAdmin bool) (userID int32) {
	t.Helper()

	q := sqlf.Sprintf("INSERT INTO users (username, site_admin) VALUES (%s, %t) RETURNING id", name, isAdmin)
	err := db.QueryRowContext(context.Background(), q.Query(sqlf.PostgresBindVar), q.Args()...).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	return userID
}
