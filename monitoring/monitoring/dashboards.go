package monitoring

import (
	"strings"
	"unicode"

	"github.com/grafana-tools/sdk"
)

// UnitType for controlling the unit type display on graphs.
type UnitType string

// short returns the short string description of the unit, for qualifying a
// number of this unit type as human-readable.
func (u UnitType) short() string {
	switch u {
	case Number, "":
		return ""
	case Milliseconds:
		return "ms"
	case Seconds:
		return "s"
	case Percentage:
		return "%"
	case Bytes:
		return "B"
	case BitsPerSecond:
		return "bps"
	case ReadsPerSecond:
		return "rps"
	case WritesPerSecond:
		return "wps"
	default:
		panic("never here")
	}
}

// From https://sourcegraph.com/github.com/grafana/grafana@b63b82976b3708b082326c0b7d42f38d4bc261fa/-/blob/packages/grafana-data/src/valueFormats/categories.ts#L23
const (
	// Number is the default unit type.
	Number UnitType = "short"

	// Milliseconds for representing time.
	Milliseconds UnitType = "ms"

	// Seconds for representing time.
	Seconds UnitType = "s"

	// Percentage in the range of 0-100.
	Percentage UnitType = "percent"

	// Bytes in IEC (1024) format, e.g. for representing storage sizes.
	Bytes UnitType = "bytes"

	// BitsPerSecond, e.g. for representing network and disk IO.
	BitsPerSecond UnitType = "bps"

	// ReadsPerSecond, e.g for representing disk IO.
	ReadsPerSecond = "rps"

	// WritesPerSecond, e.g for representing disk IO.
	WritesPerSecond = "wps"
)

// setPanelSize is a helper to set a panel's size.
func setPanelSize(p *sdk.Panel, width, height int) {
	p.GridPos.W = &width
	p.GridPos.H = &height
}

// setPanelSize is a helper to set a panel's position.
func setPanelPos(p *sdk.Panel, x, y int) {
	p.GridPos.X = &x
	p.GridPos.Y = &y
}

// observablePanelID generates a panel ID unique per dashboard for an observable at a
// given group and row.
func observablePanelID(groupIndex, rowIndex, observableIndex int) uint {
	// by default, Grafana generates panel IDs starting at 0 for panels not assigned an ID.
	// to avoid conflicts, we start generated panel IDs at large number.
	const baseGeneratedPanelID = 100000
	return uint(baseGeneratedPanelID +
		(groupIndex * 100) +
		(rowIndex * 10) +
		(observableIndex * 1))
}

// isValidGrafanaUID checks if the given string is a valid UID for entry into a Grafana dashboard. This is
// primarily used in the URL, e.g. /-/debug/grafana/d/syntect-server/<UID> and allows us to have
// static URLs we can document like:
//
// 	Go to https://sourcegraph.example.com/-/debug/grafana/d/syntect-server/syntect-server
//
// Instead of having to describe all the steps to navigate there because the UID is random.
func isValidGrafanaUID(s string) bool {
	if s != strings.ToLower(s) {
		return false
	}
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-') {
			return false
		}
	}
	return true
}
