package db

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/internal/secret"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Verbose() {
		log15.Root().SetHandler(log15.DiscardHandler())
	}

	// We want to first test everything works without any encryption.
	exitCode := m.Run()
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	// Then we want to make sure everything still works with real encryption in place.
	fmt.Println("Running tests for the second time with encryption")
	secret.MockDefaultEncryptor()
	os.Exit(m.Run())
}
