package clientclient

// Location is an LSP-like location scoped to a dump.
type Location struct {
	DumpID int
	Path   string `json:"path"`
	Range  Range  `json:"range"`
}

// Range is an inclusive bounds within a file.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position is a unique position within a file.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// MonikerData describes a moniker within a dump.
type MonikerData struct {
	Kind                 string `json:"kind"`
	Scheme               string `json:"scheme"`
	Identifier           string `json:"identifier"`
	PackageInformationID string `json:"packageInformationId"`
}

// PackageInformationData describes a package within a package manager system.
type PackageInformationData struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Diagnostic describes diagnostic information attached to a location within a
// particular dump.
type Diagnostic struct {
	DumpID         int
	Path           string `json:"path"`
	Severity       int    `json:"severity"`
	Code           string `json:"code"`
	Message        string `json:"message"`
	Source         string `json:"source"`
	StartLine      int    `json:"startLine"`
	StartCharacter int    `json:"startCharacter"`
	EndLine        int    `json:"endLine"`
	EndCharacter   int    `json:"endCharacter"`
}

// CodeIntelligenceRange pairs a range with its definitions, reference, and hover text.
type CodeIntelligenceRange struct {
	Range       Range      `json:"range"`
	Definitions []Location `json:"definitions"`
	References  []Location `json:"references"`
	HoverText   string     `json:"hoverText"`
}
