package db

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
)

type MockEventLogs struct {
	LatestPing func(ctx context.Context) (*types.Event, error)
}
