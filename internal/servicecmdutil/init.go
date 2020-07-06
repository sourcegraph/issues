package servicecmdutil

import (
	"log"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/internal/debugserver"
	"github.com/sourcegraph/sourcegraph/internal/tracer"
	"go.uber.org/automaxprocs/maxprocs"
)

type opt int

const (
	NoDebugServer opt = iota // don't start a debugserver
)

// Init runs common initialization steps for commands to run services that are part of
// Sourcegraph. It should be called at the beginning of func main.
func Init(options ...opt) {
	log.SetFlags(0)

	// Tune GOMAXPROCS for kubernetes. All our binaries import this package,
	// so we tune for all of them.
	if _, err := maxprocs.Set(); err != nil {
		log15.Error("automaxprocs failed", "error", err)
	}

	tracer.Init()

	var noDebugServer bool
	for _, opt := range options {
		if opt == NoDebugServer {
			noDebugServer = true
			break
		}
	}
	if !noDebugServer {
		go debugserver.Start()
	}
}
