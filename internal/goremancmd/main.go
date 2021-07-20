// Command goremancmd exists for testing the internally vendored goreman that
// ./cmd/server uses.
package main

import (
	"log"
	"os"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/goreman"
)

func do() error {
	if len(os.Args) != 2 {
		return errors.Errorf("USAGE: %s Procfile", os.Args[0])
	}

	procfile, err := os.ReadFile(os.Args[1])
	if err != nil {
		return err
	}

	return goreman.Start(procfile, goreman.Options{
		RPCAddr: "127.0.0.1:5005",
	})
}

func main() {
	if err := do(); err != nil {
		log.Fatal(err)
	}
}
