package inference

import (
	"regexp"
	"strings"
)

// Patterns is a regular expression that matches any input that would also match
// any registered recognizer pattern.
var Patterns = func() *regexp.Regexp {
	var patterns []string
	for _, recognizer := range Recognizers {
		for _, pattern := range recognizer.Patterns() {
			patterns = append(patterns, pattern.String())
		}
	}

	return regexp.MustCompile(strings.Join(patterns, "|"))
}()

// suffixPattern creates a regular expression that matches the given literal suffix.
func suffixPattern(s string) *regexp.Regexp {
	return regexp.MustCompile(regexp.QuoteMeta(s) + "$")
}

// segmentPattern creates a regular expression that matches paths with the given segment.
func segmentPattern(s string) *regexp.Regexp {
	return regexp.MustCompile("(^|/)" + regexp.QuoteMeta(s) + "($|/)")
}
