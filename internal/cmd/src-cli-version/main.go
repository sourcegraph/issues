package main

import (
	"fmt"

	srccli "github.com/sourcegraph/sourcegraph/internal/src-cli"
)

func main() {
	fmt.Printf(srccli.MinimumVersion)
}
