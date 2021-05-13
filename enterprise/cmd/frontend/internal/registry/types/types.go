package types

import (
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/schema"
)

// Extension describes an extension in the extension registry.
//
// It is the external form of
// github.com/sourcegraph/sourcegraph/cmd/frontend/types.RegistryExtension (which is the
// internal DB type). These types should generally be kept in sync, but registry.Extension updates
// require backcompat.
type Extension struct {
	UUID        string    `json:"uuid"`
	ExtensionID string    `json:"extensionID"`
	Publisher   Publisher `json:"publisher"`
	Name        string    `json:"name"`
	Manifest    *string   `json:"manifest"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	PublishedAt time.Time `json:"publishedAt"`
	URL         string    `json:"url"`

	// RegistryURL is the URL of the remote registry that this extension was retrieved from. It is
	// not set by package registry.
	RegistryURL string `json:"-"`
}

// Publisher describes a publisher in the extension registry.
type Publisher struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// IsWorkInProgressExtension reports whether the extension manifest indicates that this extension is
// marked as a work-in-progress extension (by having a "wip": true property, or (for backcompat) a
// title that begins with "WIP:" or "[WIP]").
//
// BACKCOMPAT: This still supports titles even though extensions no longer have titles. In Feb 2019
// it will probably be safe to remove the title handling.
//
// NOTE: Keep this pattern in sync with WorkInProgressExtensionTitlePostgreSQLPattern.
func IsWorkInProgressExtension(manifest *string) bool {
	if manifest == nil {
		// Extensions with no manifest (== no releases published yet) are considered
		// work-in-progress.
		return true
	}

	var result struct {
		schema.SourcegraphExtensionManifest
		Title string
	}
	if err := jsonc.Unmarshal(*manifest, &result); err != nil {
		// An extension whose manifest fails to parse is problematic for other reasons (and an error
		// will be displayed), but it isn't helpful to also consider it work-in-progress.
		return false
	}

	return result.Wip || strings.HasPrefix(result.Title, "WIP:") || strings.HasPrefix(result.Title, "[WIP]")
}

// WorkInProgressExtensionTitlePostgreSQLPattern is the PostgreSQL "SIMILAR TO" pattern that matches
// the extension manifest's "title" property. See
// https://www.postgresql.org/docs/9.3/functions-matching.html.
//
// NOTE: Keep this pattern in sync with IsWorkInProgressExtension.
const WorkInProgressExtensionTitlePostgreSQLPattern = `(\[WIP]|WIP:)%`

// RegistryName returns the registry name given its URL.
func RegistryName(registry *url.URL) string {
	return registry.Host
}

// ParseExtensionQuery parses an extension registry query consisting of terms and the operators
// `category:"My category"`, `tag:"mytag"`, #installed, #enabled, and #disabled.
//
// This is an intentionally simple, unoptimized parser.
func ParseExtensionQuery(q string) (text, category, tag string) {
	// Tokenize.
	var lastQuote rune
	tokens := strings.FieldsFunc(q, func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case c == '"' || c == '\'':
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	})

	unquoteValue := func(s string) string {
		return strings.Trim(s, `"'`)
	}

	var textTokens []string
	for _, tok := range tokens {
		if strings.HasPrefix(tok, "category:") {
			category = unquoteValue(strings.TrimPrefix(tok, "category:"))
		} else if strings.HasPrefix(tok, "tag:") {
			tag = unquoteValue(strings.TrimPrefix(tok, "tag:"))
		} else if tok == "#installed" || tok == "#enabled" || tok == "#disabled" {
			// Ignore so that the client can implement these in post-processing.
		} else {
			textTokens = append(textTokens, tok)
		}
	}
	return strings.Join(textTokens, " "), category, tag
}
