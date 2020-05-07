package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sourcegraph/sourcegraph/cmd/precise-code-intel-api-server/internal/resetter"
	"github.com/sourcegraph/sourcegraph/cmd/precise-code-intel-api-server/internal/server"
	bundles "github.com/sourcegraph/sourcegraph/internal/codeintel/bundles/client"
	"github.com/sourcegraph/sourcegraph/internal/codeintel/db"
	"github.com/sourcegraph/sourcegraph/internal/codeintel/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/debugserver"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/tracer"
)

func main() {
	env.Lock()
	env.HandleHelpFlag()
	tracer.Init()

	var (
		bundleManagerURL = mustGet(rawBundleManagerURL, "PRECISE_CODE_INTEL_BUNDLE_MANAGER_URL")
		resetInterval    = mustParseInterval(rawResetInterval, "PRECISE_CODE_INTEL_RESET_INTERVAL")
	)

	db := mustInitializeDatabase()

	host := ""
	if env.InsecureDev {
		host = "127.0.0.1"
	}

	serverInst := server.New(server.ServerOpts{
		Host:                host,
		Port:                3186,
		DB:                  db,
		BundleManagerClient: bundles.New(bundleManagerURL),
		GitserverClient:     gitserver.DefaultClient,
	})

	resetterMetrics := resetter.NewResetterMetrics()
	resetterMetrics.MustRegister(prometheus.DefaultRegisterer)

	uploadResetterInst := resetter.NewUploadResetter(resetter.UploadResetterOpts{
		DB:            db,
		ResetInterval: resetInterval,
		Metrics:       resetterMetrics,
	})

	go serverInst.Start()
	go uploadResetterInst.Run()
	go debugserver.Start()
	waitForSignal()
}

func mustInitializeDatabase() db.DB {
	postgresDSN := conf.Get().ServiceConnections.PostgresDSN
	conf.Watch(func() {
		if newDSN := conf.Get().ServiceConnections.PostgresDSN; postgresDSN != newDSN {
			log.Fatalf("Detected repository DSN change, restarting to take effect: %s", newDSN)
		}
	})

	db, err := db.New(postgresDSN)
	if err != nil {
		log.Fatalf("failed to initialize db store: %s", err)
	}

	return db
}

func waitForSignal() {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGHUP)

	for i := 0; i < 2; i++ {
		<-signals
	}

	os.Exit(0)
}
