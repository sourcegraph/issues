package graphqlbackend

import (
	"fmt"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

// This constant defines the cursor prefix, which disambiguates a repository
// cursor from other types of cursors in the system.
const repositoryCursorKind = "RepositoryCursor"

// A repositoryCursor can be provided to a `repositories` query for efficient
// cursor-based pagination (vs. LIMIT/OFFSET).
type repositoryCursor struct {
	Column    string
	Value     string
	Direction string
}

// marshalRepositoryCursor marshals a repository pagination cursor.
func marshalRepositoryCursor(cursor *repositoryCursor) string {
	return string(relay.MarshalID(repositoryCursorKind, cursor))
}

// unmarshalRepositoryCursor unmarshals a repository pagination cursor.
func unmarshalRepositoryCursor(cursor *string) (*repositoryCursor, error) {
	if cursor == nil {
		return nil, nil
	}
	if kind := relay.UnmarshalKind(graphql.ID(*cursor)); kind != repositoryCursorKind {
		return nil, fmt.Errorf("cannot unmarshal repository cursor type: %q", kind)
	}
	var spec *repositoryCursor
	if err := relay.UnmarshalSpec(graphql.ID(*cursor), &spec); err != nil {
		return nil, err
	}
	return spec, nil
}
