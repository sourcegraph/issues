package registry

import (
	"strings"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry/types"
)

// FilterRegistryExtensions returns the subset of extensions that match the query. It does not
// modify its arguments.
func FilterRegistryExtensions(extensions []*types.Extension, query string) []*types.Extension {
	if query == "" {
		return extensions
	}

	query = strings.ToLower(query)
	var keep []*types.Extension
	for _, x := range extensions {
		if strings.Contains(strings.ToLower(x.ExtensionID), query) {
			keep = append(keep, x)
		}
	}
	return keep
}

// FindRegistryExtension returns the first (and, hopefully, only, although that's not enforced)
// extension whose field matches the given value, or nil if none match.
func FindRegistryExtension(extensions []*types.Extension, field, value string) *types.Extension {
	match := func(x *types.Extension) bool {
		switch field {
		case "uuid":
			return x.UUID == value
		case "extensionID":
			return x.ExtensionID == value
		default:
			panic("unexpected field: " + field)
		}
	}

	for _, x := range extensions {
		if match(x) {
			return x
		}
	}
	return nil
}
