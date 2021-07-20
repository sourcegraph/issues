package search

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-enry/go-enry/v2"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/search/filter"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
)

// unionRegexp separates values with a | operator to create a string
// representing a union of regexp patterns.
func unionRegexp(values []string) string {
	if len(values) == 0 {
		// As a regular expression, "()" and "" are equivalent so this
		// condition wouldn't ordinarily be needed to distinguish these
		// values. But, our internal search engine assumes that ""
		// implies "no regexp" (no values), while "()" implies "match
		// empty regexp" (all values) for file patterns.
		return ""
	}
	if len(values) == 1 {
		// Cosmetic format for regexp value, wherever this happens to be
		// pretty printed.
		return values[0]
	}
	return "(" + strings.Join(values, ")|(") + ")"
}

// langToFileRegexp converts a lang: parameter to its corresponding file
// patterns for file filters. The lang value must be valid, cf. validate.go
func langToFileRegexp(lang string) string {
	lang, _ = enry.GetLanguageByAlias(lang) // Invariant: lang is valid.
	extensions := enry.GetLanguageExtensions(lang)
	patterns := make([]string, len(extensions))
	for i, e := range extensions {
		// Add `\.ext$` pattern to match files with the given extension.
		patterns[i] = regexp.QuoteMeta(e) + "$"
	}
	return unionRegexp(patterns)
}

func mapSlice(values []string, f func(string) string) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = f(v)
	}
	return result
}

func IncludeExcludeValues(q query.Basic, field string) (include, exclude []string) {
	q.VisitParameter(field, func(v string, negated bool, _ query.Annotation) {
		if negated {
			exclude = append(exclude, v)
		} else {
			include = append(include, v)
		}
	})
	return include, exclude
}

func count(q query.Basic, p Protocol) int {
	if count := q.GetCount(); count != "" {
		v, _ := strconv.Atoi(count) // Invariant: count is validated.
		return v
	}

	if q.IsStructural() {
		return DefaultMaxSearchResults
	}

	switch p {
	case Batch:
		return DefaultMaxSearchResults
	case Streaming:
		return DefaultMaxSearchResultsStreaming
	}
	panic("unreachable")
}

type Protocol int

const (
	Streaming Protocol = iota
	Batch
)

// ToTextPatternInfo converts a an atomic query to internal values that drive
// text search. An atomic query is a Basic query where the Pattern is either
// nil, or comprises only one Pattern node (hence, an atom, and not an
// expression). See TextPatternInfo for the values it computes and populates.
func ToTextPatternInfo(q query.Basic, p Protocol, transform query.BasicPass) *TextPatternInfo {
	q = transform(q)
	// Handle file: and -file: filters.
	filesInclude, filesExclude := IncludeExcludeValues(q, query.FieldFile)
	// Handle lang: and -lang: filters.
	langInclude, langExclude := IncludeExcludeValues(q, query.FieldLang)
	filesInclude = append(filesInclude, mapSlice(langInclude, langToFileRegexp)...)
	filesExclude = append(filesExclude, mapSlice(langExclude, langToFileRegexp)...)
	filesReposMustInclude, filesReposMustExclude := IncludeExcludeValues(q, query.FieldRepoHasFile)
	selector, _ := filter.SelectPathFromString(q.FindValue(query.FieldSelect)) // Invariant: select is validated
	count := count(q, p)

	// Ugly assumption: for a literal search, the IsRegexp member of
	// TextPatternInfo must be set true. The logic assumes that a literal
	// pattern is an escaped regular expression.
	isRegexp := q.IsLiteral() || q.IsRegexp()

	var pattern string
	if p, ok := q.Pattern.(query.Pattern); ok {
		if q.IsLiteral() {
			// Escape regexp meta characters if this pattern should be treated literally.
			pattern = regexp.QuoteMeta(p.Value)
		} else {
			pattern = p.Value
		}
	}

	if q.Pattern == nil {
		// For compatibility: A nil pattern implies isRegexp is set to
		// true. This has no effect on search logic.
		isRegexp = true
	}

	negated := false
	if p, ok := q.Pattern.(query.Pattern); ok {
		negated = p.Negated
	}

	return &TextPatternInfo{
		// Values dependent on pattern atom.
		IsRegExp:        isRegexp,
		IsStructuralPat: q.IsStructural(),
		IsCaseSensitive: q.IsCaseSensitive(),
		FileMatchLimit:  int32(count),
		Pattern:         pattern,
		IsNegated:       negated,

		// Values dependent on parameters.
		IncludePatterns:              filesInclude,
		ExcludePattern:               unionRegexp(filesExclude),
		FilePatternsReposMustInclude: filesReposMustInclude,
		FilePatternsReposMustExclude: filesReposMustExclude,
		Languages:                    langInclude,
		PathPatternsAreCaseSensitive: q.IsCaseSensitive(),
		CombyRule:                    q.FindValue(query.FieldCombyRule),
		Index:                        q.Index(),
		Select:                       selector,
	}
}

func TimeoutDuration(b query.Basic) time.Duration {
	d := DefaultTimeout
	maxTimeout := time.Duration(SearchLimits(conf.Get()).MaxTimeoutSeconds) * time.Second
	timeout := b.GetTimeout()
	if timeout != nil {
		d = *timeout
	} else if b.GetCount() != "" {
		// If `count:` is set but `timeout:` is not explicitly set, use the max timeout
		d = maxTimeout
	}
	if d > maxTimeout {
		d = maxTimeout
	}
	return d
}
