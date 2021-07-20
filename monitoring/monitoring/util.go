package monitoring

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
)

// upperFirst returns s with an uppercase first rune.
func upperFirst(s string) string {
	return strings.ToUpper(string([]rune(s)[0])) + string([]rune(s)[1:])
}

// withPeriod returns s ending with a period.
func withPeriod(s string) string {
	if !strings.HasSuffix(s, ".") {
		return s + "."
	}
	return s
}

// StringPtr converts a string value to a pointer, useful for setting fields in some APIs.
func StringPtr(s string) *string { return &s }

// boolPtr converts a boolean value to a pointer, useful for setting fields in some APIs.
func boolPtr(b bool) *bool { return &b }

// IntPtr converts an int64 value to a pointer, useful for setting fields in some APIs.
func Int64Ptr(i int64) *int64 { return &i }

// toMarkdown converts a Go string to Markdown, and optionally converts it to a list item if requested by forceList.
func toMarkdown(m string, forceList bool) (string, error) {
	m = strings.TrimPrefix(m, "\n")

	// Replace single quotes with backticks.
	// Replace escaped single quotes with single quotes.
	m = strings.ReplaceAll(m, `\'`, `$ESCAPED_SINGLE_QUOTE`)
	m = strings.ReplaceAll(m, `'`, "`")
	m = strings.ReplaceAll(m, `$ESCAPED_SINGLE_QUOTE`, "'")

	// Unindent based on the indention of the last line.
	lines := strings.Split(m, "\n")
	baseIndention := lines[len(lines)-1]
	if strings.TrimSpace(baseIndention) == "" {
		if strings.Contains(baseIndention, " ") {
			return "", errors.New("go string literal indention must be tabs")
		}
		indentionLevel := strings.Count(baseIndention, "\t")
		removeIndention := strings.Repeat("\t", indentionLevel+1)
		for i, l := range lines[:len(lines)-1] {
			trimmedLine := strings.TrimPrefix(l, removeIndention)
			if l != "" && l == trimmedLine {
				return "", errors.Errorf("inconsistent indention (line %d %q expected to start with %q)", i, l, removeIndention)
			}
			lines[i] = trimmedLine
		}
		m = strings.Join(lines[:len(lines)-1], "\n")
	}

	if forceList {
		// If result is not a list, make it a list, so we can add items.
		if !strings.HasPrefix(m, "-") && !strings.HasPrefix(m, "*") {
			m = fmt.Sprintf("- %s", m)
		}
	}

	return m, nil
}
