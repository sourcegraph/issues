// Package siteid provides access to the site ID, a stable identifier for the current
// Sourcegraph site.
//
// All servers that are part of the same logical Sourcegraph site have the same site ID
// (although it is possible for an admin to misconfigure the servers so that this is not
// true).
//
// The "site ID" was formerly known as the "app ID".
package siteid

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/db/globalstatedb"
)

var (
	inited bool
	siteID string

	fatalln = log.Fatalln // overridden in tests
)

// Init reads (or generates) the site ID. This func must be called exactly once before
// Get can be called.
func Init() {
	if inited {
		panic("siteid: already initialized")
	}

	if v := os.Getenv("TRACKING_APP_ID"); v != "" {
		// Legacy way of specifying site ID.
		//
		// TODO(dadlerj): remove this
		siteID = v
	} else {
		// Site ID is retrieved from the database (where it might be created automatically
		// if it doesn't yet exist.)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		globalState, err := globalstatedb.Get(ctx)
		if err != nil {
			fatalln("Error initializing global state:", err)
		}
		siteID = globalState.SiteID
	}

	inited = true
}

// Get returns the site ID.
//
// Get may only be called after Init has been called.
func Get() string {
	if !inited {
		panic("siteid: not yet initialized")
	}
	return siteID
}
