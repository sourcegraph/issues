package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/worker/shared"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/worker/internal/codeintel"
	eiauthz "github.com/sourcegraph/sourcegraph/enterprise/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
)

func main() {
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	if debug {
		log.Println("enterprise edition")
	}

	go setAuthzProviders()

	shared.Start(map[string]shared.Job{
		"codeintel-commitgraph": codeintel.NewCommitGraphJob(),
		"codeintel-janitor":     codeintel.NewJanitorJob(),
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
