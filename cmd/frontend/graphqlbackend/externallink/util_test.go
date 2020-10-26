package externallink

import (
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
)

func resetMocks() {
	repoupdater.MockRepoLookup = nil
	db.Mocks = db.MockStores{}
	backend.Mocks = backend.MockServices{}
}
