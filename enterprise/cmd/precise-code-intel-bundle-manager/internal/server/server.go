package server

import (
	"database/sql"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/persistence/cache"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/httpserver"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

const Port = 3187

type Server struct {
	bundleDir          string
	storeCache         cache.StoreCache
	codeIntelDB        *sql.DB
	observationContext *observation.Context
}

func New(bundleDir string, storeCache cache.StoreCache, codeIntelDB *sql.DB, observationContext *observation.Context) goroutine.BackgroundRoutine {
	server := &Server{
		bundleDir:          bundleDir,
		storeCache:         storeCache,
		codeIntelDB:        codeIntelDB,
		observationContext: observationContext,
	}

	return httpserver.New(Port, server.setupRoutes)
}
