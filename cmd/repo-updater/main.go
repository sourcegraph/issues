// Command repo-updater periodically updates repositories configured in site configuration and serves repository
// metadata from multiple external code hosts.
package main

import (
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/shared"
	"github.com/sourcegraph/sourcegraph/internal/servicecmdutil"
)

func main() {
	servicecmdutil.Init(servicecmdutil.NoDebugServer) // registers a custom debugserver handler
	shared.Main(nil)
}
