package resolver

import (
	"sync"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/schema"
)

// validateExtensionManifest validates a JSON extension manifest for syntax.
//
// TODO(sqs): Also validate it against the JSON Schema.
func validateExtensionManifest(text string) error {
	var o interface{}
	return jsonc.Unmarshal(text, &o)
}

// extensionManifest implements the GraphQL type ExtensionManifest.
type extensionManifest struct {
	raw string

	// cache result because it is used by multiple fields
	once   sync.Once
	result *schema.SourcegraphExtensionManifest
	err    error
}

// NewExtensionManifest creates a new resolver for the GraphQL type ExtensionManifest with the given
// raw contents of an extension manifest.
func NewExtensionManifest(raw *string) graphqlbackend.ExtensionManifest {
	if raw == nil {
		return nil
	}
	return &extensionManifest{raw: *raw}
}

func (r *extensionManifest) parse() (*schema.SourcegraphExtensionManifest, error) {
	r.once.Do(func() {
		r.err = jsonc.Unmarshal(r.raw, &r.result)
	})
	return r.result, r.err
}

func (r *extensionManifest) Raw() string { return r.raw }

func (r *extensionManifest) Description() (*string, error) {
	parsed, err := r.parse()
	if parsed == nil || err != nil {
		return nil, err
	}
	if parsed.Description == "" {
		return nil, nil
	}
	return &parsed.Description, nil
}

func (r *extensionManifest) BundleURL() (*string, error) {
	parsed, err := r.parse()
	if parsed == nil || err != nil {
		return nil, err
	}
	if parsed.Url == "" {
		return nil, nil
	}
	return &parsed.Url, nil
}
