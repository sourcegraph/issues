package types

import (
	"time"
)

type Worker struct {
	ID         int64
	Name       string
	Token      *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastSeenAt time.Time
}
