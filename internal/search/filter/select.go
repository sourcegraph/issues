package filter

import (
	"strings"

	"github.com/cockroachdb/errors"
)

const (
	Commit     = "commit"
	Content    = "content"
	File       = "file"
	Repository = "repo"
	Symbol     = "symbol"
)

// SelectPath represents a parsed and validated select value
type SelectPath []string

func (sp SelectPath) String() string {
	return strings.Join(sp, ".")
}

// Root is the top-level result type that is being selected.
// Returns an empty string if SelectPath is empty
func (sp SelectPath) Root() string {
	if len(sp) > 0 {
		return sp[0]
	}
	return ""
}

type object map[string]object

var validSelectors = object{
	Commit: object{
		"diff": object{
			"added":   nil,
			"removed": nil,
		},
	},
	Content: nil,
	File: {
		"directory": nil,
		"path":      nil,
	},
	Repository: nil,
	Symbol: object{
		/* cf. SymbolKind https://microsoft.github.io/language-server-protocol/specification */
		"file":           nil,
		"module":         nil,
		"namespace":      nil,
		"package":        nil,
		"class":          nil,
		"method":         nil,
		"property":       nil,
		"field":          nil,
		"constructor":    nil,
		"enum":           nil,
		"interface":      nil,
		"function":       nil,
		"variable":       nil,
		"constant":       nil,
		"string":         nil,
		"number":         nil,
		"boolean":        nil,
		"array":          nil,
		"object":         nil,
		"key":            nil,
		"null":           nil,
		"enum-member":    nil,
		"struct":         nil,
		"event":          nil,
		"operator":       nil,
		"type-parameter": nil,
	},
}

func SelectPathFromString(s string) (SelectPath, error) {
	fields := strings.Split(s, ".")
	cur := validSelectors
	for _, field := range fields {
		child, ok := cur[field]
		if !ok {
			return SelectPath{}, errors.Errorf("invalid field %q on select path %q", field, s)
		}
		cur = child
	}
	return SelectPath(fields), nil
}
