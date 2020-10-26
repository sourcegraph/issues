// Command minversion ensures users are running the minimum required Go
// version. If not, it will exit with a non-zero exit code.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/mcuadros/go-version"
)

func main() {
	// This should be the lowest version our toolchain supports in development
	// mode, not necessarily latest patch version of go. Every time you bump
	// this you are forcing our devs to update, so we need a good reason
	// (tools stop working, creates changes in version tracked files,
	// etc). The version should be satisfiable by the latest go in brew.
	//
	// Note: This is just for development, our images are built in CI with the
	// version specified in .tool-versions in the root of our repository.
	minimumVersion := "1.14"
	rawVersion := runtime.Version()
	versionNumber := strings.TrimPrefix(rawVersion, "go")
	minimumVersionMet := version.Compare(minimumVersion, versionNumber, "<=")
	if !minimumVersionMet {
		fmt.Printf("Go version %s or newer must be used; found: %s\n", minimumVersion, versionNumber)
		os.Exit(1) // minimum version not met means non-zero exit code
	}
}
