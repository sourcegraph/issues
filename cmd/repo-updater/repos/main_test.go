package repos

import (
	"flag"
	"os"
	"regexp"
	"testing"

	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/secret"
)

var updateRegex = flag.String("update", "", "Update testdata of tests matching the given regex")

func update(name string) bool {
	if updateRegex == nil || *updateRegex == "" {
		return false
	}
	return regexp.MustCompile(*updateRegex).MatchString(name)
}

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Verbose() {
		log15.Root().SetHandler(log15.DiscardHandler())
	}

	// NOTE: We only run tests with real encryption because calling m.Run twice will panic with duplicate metrics.
	secret.MockDefaultEncryptor()
	os.Exit(m.Run())
}
