package shared

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/authz"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/enterprise"
	frontend_shared "github.com/sourcegraph/sourcegraph/cmd/frontend/shared"
	gitserver_shared "github.com/sourcegraph/sourcegraph/cmd/gitserver/shared"
	repoupdater_shared "github.com/sourcegraph/sourcegraph/cmd/repo-updater/shared"
	searcher_shared "github.com/sourcegraph/sourcegraph/cmd/searcher/shared"
)

// Main is the single-program command function, which is shared between Sourcegraph's open-source
// and enterprise variant.
func Main() {
	flag.Parse()
	log.SetFlags(0)

	os.Setenv("CONFIGURATION_MODE", "server")
	os.Setenv("SRC_REPOS_DIR", "/tmp/sourcegraph-repos")
	os.Setenv("SRC_HTTP_ADDR", ":7077")
	os.Setenv("SRC_HTTP_ADDR_INTERNAL", ":7078")
	os.Setenv("SRC_FRONTEND_INTERNAL", "localhost:7078")
	os.Setenv("SRC_LOG_LEVEL", "info")
	os.Setenv("REDIS_ENDPOINT", "127.0.0.1:6379")
	os.Setenv("SRC_GIT_SERVERS", "127.0.0.1:3178")
	os.Setenv("SRC_REPOS_DIR", "/tmp/sourcegraph-repos")
	os.Setenv("LOGO", "t")

	if err := RunAllSingleProcess(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func RunAllSingleProcess(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, len(programs))
	for name, p := range programs {
		if name != "frontend" {
			continue // TODO(sqs)
		}
		go func(name string, program program) {
			log.Printf("RUN %s", name)
			err := program.main(ctx)
			if err == nil {
				panic(fmt.Sprintf("program terminated: %s", name))
			}
			if err == context.Canceled {
				err = nil
			}
			errChan <- err
		}(name, p)
	}

	var errs error
	for i := 0; i < len(errChan); i++ {
		if err := <-errChan; err != nil && err != context.Canceled {
			cancel()
			errs = multierror.Append(errs, err)
		}
	}
	close(errChan)
	return nil
}

type program struct {
	main func(context.Context) error
}

var programs = map[string]program{
	"frontend": {

		main: func(ctx context.Context) error {
			// Set dummy authz provider to unblock channel for checking permissions in GraphQL APIs.
			// See https://github.com/sourcegraph/sourcegraph/issues/3847 for details.
			authz.SetProviders(true, []authz.Provider{})

			frontend_shared.Main(func() enterprise.Services {
				return enterprise.DefaultServices()
			})
			return nil
		},
	},
	"gitserver": {
		main: func(ctx context.Context) error {
			gitserver_shared.Main()
			return nil
		},
	},
	"searcher": {
		main: func(ctx context.Context) error {
			searcher_shared.Main()
			return nil
		},
	},
	"repo-updater": {
		main: func(ctx context.Context) error {
			repoupdater_shared.Main(nil) // TODO(sqs): missing enterprise
			return nil
		},
	},
	// TODO(sqs): no zoekt, query-runner
}
