// Command frontend contains the enterprise frontend implementation.
//
// It wraps the open source frontend command and merely injects a few
// proprietary things into it via e.g. blank/underscore imports in this file
// which register side effects with the frontend package.
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/authz"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/db"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/enterprise"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/shared"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	_ "github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/auth"
	eauthz "github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/authz"
	authzResolvers "github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/authz/resolvers"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/dotcom/productsubscription"
	_ "github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/licensing"
	_ "github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	campaignsResolvers "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/resolvers"
	codeintelapi "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/api"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	codeintelgitserver "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
	codeintelhttpapi "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/httpapi"
	codeintelresolvers "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/resolvers"
	codeintelgqlresolvers "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/resolvers/graphql"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/globalstatedb"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/servicecmdutil"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

func main() {
	servicecmdutil.Init()

	shared.Main(func() enterprise.Services {
		ctx := context.Background()
		enterpriseServices := enterprise.DefaultServices()

		initLicensing()
		initAuthz(ctx, &enterpriseServices)
		initCampaigns(ctx, &enterpriseServices)
		initCodeIntel(&enterpriseServices)

		return enterpriseServices
	})
}

var msResolutionClock = func() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

func initLicensing() {
	// TODO(efritz) - de-globalize assignments in this function

	// Enforce the license's max user count by preventing the creation of new users when the max is
	// reached.
	db.Users.PreCreateUser = licensing.NewPreCreateUserHook(&usersStore{})

	// Make the Site.productSubscription.productNameWithBrand GraphQL field (and other places) use the
	// proper product name.
	graphqlbackend.GetProductNameWithBrand = licensing.ProductNameWithBrand

	// Make the Site.productSubscription.actualUserCount and Site.productSubscription.actualUserCountDate
	// GraphQL fields return the proper max user count and timestamp on the current license.
	graphqlbackend.ActualUserCount = licensing.ActualUserCount
	graphqlbackend.ActualUserCountDate = licensing.ActualUserCountDate

	noLicenseMaximumAllowedUserCount := licensing.NoLicenseMaximumAllowedUserCount
	graphqlbackend.NoLicenseMaximumAllowedUserCount = &noLicenseMaximumAllowedUserCount

	noLicenseWarningUserCount := licensing.NoLicenseWarningUserCount
	graphqlbackend.NoLicenseWarningUserCount = &noLicenseWarningUserCount

	// Make the Site.productSubscription GraphQL field return the actual info about the product license,
	// if any.
	graphqlbackend.GetConfiguredProductLicenseInfo = func() (*graphqlbackend.ProductLicenseInfo, error) {
		info, err := licensing.GetConfiguredProductLicenseInfo()
		if info == nil || err != nil {
			return nil, err
		}
		return &graphqlbackend.ProductLicenseInfo{
			TagsValue:      info.Tags,
			UserCountValue: info.UserCount,
			ExpiresAtValue: info.ExpiresAt,
		}, nil
	}

	goroutine.Go(func() {
		licensing.StartMaxUserCount(&usersStore{})
	})
	if envvar.SourcegraphDotComMode() {
		goroutine.Go(productsubscription.StartCheckForUpcomingLicenseExpirations)
	}
}

func initAuthz(ctx context.Context, enterpriseServices *enterprise.Services) {
	eauthz.Init(dbconn.Global, msResolutionClock)

	go func() {
		t := time.NewTicker(5 * time.Second)
		for range t.C {
			allowAccessByDefault, authzProviders, _, _ :=
				eauthz.ProvidersFromConfig(ctx, conf.Get(), db.ExternalServices, dbconn.Global)
			authz.SetProviders(allowAccessByDefault, authzProviders)
		}
	}()

	enterpriseServices.AuthzResolver = authzResolvers.NewResolver(dbconn.Global, func() time.Time {
		return time.Now().UTC().Truncate(time.Microsecond)
	})
}

func initCampaigns(ctx context.Context, enterpriseServices *enterprise.Services) {
	globalState, err := globalstatedb.Get(ctx)
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}

	campaignsStore := campaigns.NewStoreWithClock(dbconn.Global, msResolutionClock)

	// Migrate all patches in the database to cache their diff stats.
	// Since we validate each Patch's diff before we store it in the database,
	// this migration should never fail, except in exceptional circumstances
	// (database not reachable), in which case it's okay to exit.
	//
	// This can be removed in 3.19.

	if err := campaigns.MigratePatchesWithoutDiffStats(ctx, campaignsStore); err != nil {
		log.Fatalf("FATAL: Migrating patches without diff stats: %v", err)
	}

	repositories := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	enterpriseServices.CampaignsResolver = campaignsResolvers.NewResolver(dbconn.Global)
	enterpriseServices.GithubWebhook = campaigns.NewGitHubWebhook(campaignsStore, repositories, msResolutionClock)
	enterpriseServices.BitbucketServerWebhook = campaigns.NewBitbucketServerWebhook(
		campaignsStore,
		repositories,
		msResolutionClock,
		"sourcegraph-"+globalState.SiteID,
	)
}

var bundleManagerURL = env.Get("PRECISE_CODE_INTEL_BUNDLE_MANAGER_URL", "", "HTTP address for internal LSIF bundle manager server.")
var rawHunkCacheSize = env.Get("PRECISE_CODE_INTEL_HUNK_CACHE_CAPACITY", "1000", "Maximum number of git diff hunk objects that can be loaded into the hunk cache at once.")

func initCodeIntel(enterpriseServices *enterprise.Services) {
	if bundleManagerURL == "" {
		log.Fatalf("invalid value for PRECISE_CODE_INTEL_BUNDLE_MANAGER_URL: no value supplied")
	}

	hunkCacheSize, err := strconv.ParseInt(rawHunkCacheSize, 10, 64)
	if err != nil {
		log.Fatalf("invalid int %q for PRECISE_CODE_INTEL_HUNK_CACHE_CAPACITY: %s", rawHunkCacheSize, err)
	}

	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	store := store.NewObserved(store.NewWithHandle(dbconn.Global), observationContext)
	bundleManagerClient := bundles.New(bundleManagerURL)
	api := codeintelapi.NewObserved(codeintelapi.New(store, bundleManagerClient, codeintelgitserver.DefaultClient), observationContext)
	hunkCache, err := codeintelresolvers.NewHunkCache(int(hunkCacheSize))
	if err != nil {
		log.Fatalf("failed to initialize hunk cache: %s", err)
	}

	enterpriseServices.CodeIntelResolver = codeintelgqlresolvers.NewResolver(codeintelresolvers.NewResolver(
		store,
		bundleManagerClient,
		api,
		hunkCache,
	))

	enterpriseServices.NewCodeIntelUploadHandler = func(internal bool) http.Handler {
		return codeintelhttpapi.NewUploadHandler(store, bundleManagerClient, internal)
	}
}

type usersStore struct{}

func (usersStore) Count(ctx context.Context) (int, error) {
	return db.Users.Count(ctx, nil)
}
