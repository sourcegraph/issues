package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/cmd/worker/shared"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/worker/internal/codeintel"
	eiauthz "github.com/sourcegraph/sourcegraph/enterprise/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/versions"
	"github.com/sourcegraph/sourcegraph/internal/repos"
)

func main() {
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	if debug {
		log.Println("enterprise edition")
	}

	db, err := shared.InitDatabase()
	if err != nil {
		panic("asdf")
	}

	store := repos.NewStore(db, sql.TxOptions{Isolation: sql.LevelDefault})
	{
		m := repos.NewStoreMetrics()
		m.MustRegister(prometheus.DefaultRegisterer)
		store.Metrics = m
	}

	go setAuthzProviders()

	shared.Start(map[string]shared.Job{
		"codeintel-commitgraph":           codeintel.NewCommitGraphJob(),
		"codeintel-janitor":               codeintel.NewJanitorJob(),
		"codeintel-auto-indexing":         codeintel.NewIndexingJob(),
		"codeintel-dependency-repo-adder": codeintel.NewDependencyRepoAddingJob(),
		"codehost-version-syncing":        versions.NewSyncingJob(store),
	})
}

// setAuthProviders waits for the database to be initialized, then periodically refreshes the
// global authz providers. This changes the repositories that are visible for reads based on the
// current actor stored in an operation's context, which is likely an internal actor for many of
// the jobs configured in this service. This also enables repository update operations to fetch
// permissions from code hosts.
func setAuthzProviders() {
	db, err := shared.InitDatabase()
	if err != nil {
		return
	}

	ctx := context.Background()

	for range time.NewTicker(5 * time.Second).C {
		allowAccessByDefault, authzProviders, _, _ := eiauthz.ProvidersFromConfig(ctx, conf.Get(), database.ExternalServices(db))
		authz.SetProviders(allowAccessByDefault, authzProviders)
	}
}
