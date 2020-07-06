// Command searcher is a simple service which exposes an API to text search a
// repo at a specific commit. See the searcher package for more information.
package main

import (
	"github.com/sourcegraph/sourcegraph/cmd/searcher/shared"
	"github.com/sourcegraph/sourcegraph/internal/servicecmdutil"
)

func main() {
	servicecmdutil.Init()
	shared.Main()
}
