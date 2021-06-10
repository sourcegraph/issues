package globalstatedb

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
)

func TestGet(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	db := dbtest.NewDB(t, "")
	ctx := context.Background()
	config, err := Get(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if config.SiteID == "" {
		t.Fatal("expected site_id to be set")
	}
}
