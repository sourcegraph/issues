package repos

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/extsvc/jvmpackages/coursier"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestJvmPackagesClone(t *testing.T) {
	var c schema.JvmPackagesConnection
	c.Repositories = []string{"central"}
	x, err := coursier.FetchSources(context.Background(), &c, "junit:junit:4.13.2")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(x)
}
