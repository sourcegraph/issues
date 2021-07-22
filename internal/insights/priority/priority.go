// Package priority implements a basic priority scheme for insight query execution and ordering.
package priority

import "time"

type Priority int

const (
	Critical Priority = 1
	High     Priority = 10
	Medium   Priority = 100
	Low      Priority = 1000
)

// FromTimeInterval calculates a Priority value for an insight by deriving a value based on a time range. This value will rank more recent data points
// higher priority than older ones. This can be useful for backfilling and ensuring multiple insights backfill at roughly the same rate.
func FromTimeInterval(from time.Time, to time.Time) Priority {
	days := to.Sub(from).Hours() / 24
	return Priority(days)
}

func (p Priority) LowerBy(val int) Priority {
	return Priority(int(p) - val)
}

func (p Priority) RaiseBy(val int) Priority {
	return Priority(int(p) + val)
}

func (p Priority) Lower() Priority {
	return p.LowerBy(1)
}

func (p Priority) Raise() Priority {
	return p.RaiseBy(1)
}
