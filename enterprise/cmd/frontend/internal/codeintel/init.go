package codeintel

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/enterprise"
	codeintelapi "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/api"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
	codeintelhttpapi "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/httpapi"
	codeintelresolvers "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/resolvers"
	codeintelgqlresolvers "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/resolvers/graphql"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

var bundleManagerURL = env.Get("PRECISE_CODE_INTEL_BUNDLE_MANAGER_URL", "", "HTTP address for internal LSIF bundle manager server.")
var rawHunkCacheSize = env.Get("PRECISE_CODE_INTEL_HUNK_CACHE_CAPACITY", "1000", "Maximum number of git diff hunk objects that can be loaded into the hunk cache at once.")

func Init(ctx context.Context, enterpriseServices *enterprise.Services) error {
	if bundleManagerURL == "" {
		return fmt.Errorf("invalid value for PRECISE_CODE_INTEL_BUNDLE_MANAGER_URL: no value supplied")
	}

	hunkCacheSize, err := strconv.ParseInt(rawHunkCacheSize, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid int %q for PRECISE_CODE_INTEL_HUNK_CACHE_CAPACITY: %s", rawHunkCacheSize, err)
	}

	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	codeIntelDB := mustInitializeCodeIntelDatabase()

	store := store.NewObserved(store.NewWithDB(dbconn.Global), observationContext)
	bundleManagerClient := bundles.New(codeIntelDB, observationContext, bundleManagerURL)
	api := codeintelapi.NewObserved(codeintelapi.New(store, bundleManagerClient, gitserver.DefaultClient), observationContext)
	hunkCache, err := codeintelresolvers.NewHunkCache(int(hunkCacheSize))
	if err != nil {
		return fmt.Errorf("failed to initialize hunk cache: %s", err)
	}

	enterpriseServices.CodeIntelResolver = codeintelgqlresolvers.NewResolver(codeintelresolvers.NewResolver(
		store,
		bundleManagerClient,
		api,
		hunkCache,
	))

	newCodeIntelUploadHandler := func(internal bool) http.Handler {
		return codeintelhttpapi.NewUploadHandler(store, bundleManagerClient, internal)
	}

	enterpriseServices.NewCodeIntelUploadHandler = newCodeIntelUploadHandler

	h, err := newInternalProxyHandler(newCodeIntelUploadHandler(true))
	if err != nil {
		return err
	}

	enterpriseServices.NewCodeIntelInternalProxyHandler = h
	return nil
}

func mustInitializeCodeIntelDatabase() *sql.DB {
	postgresDSN := conf.Get().ServiceConnections.CodeIntelPostgresDSN
	conf.Watch(func() {
		if newDSN := conf.Get().ServiceConnections.CodeIntelPostgresDSN; postgresDSN != newDSN {
			log.Fatalf("detected database DSN change, restarting to take effect: %s", newDSN)
		}
	})

	db, err := dbconn.New(postgresDSN, "_codeintel")
	if err != nil {
		log.Fatalf("failed to connect to codeintel database: %s", err)
	}

	if err := dbconn.MigrateDB(db, "codeintel"); err != nil {
		log.Fatalf("failed to perform codeintel database migration: %s", err)
	}

	return db
}
