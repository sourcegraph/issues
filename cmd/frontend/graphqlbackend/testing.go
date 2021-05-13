package graphqlbackend

import (
	"sync"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

var (
	parseSchemaOnce sync.Once
	parseSchemaErr  error
	parsedSchema    *graphql.Schema
)

func mustParseGraphQLSchema(t *testing.T) *graphql.Schema {
	t.Helper()

	parseSchemaOnce.Do(func() {
		parsedSchema, parseSchemaErr = NewSchema(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	})
	if parseSchemaErr != nil {
		t.Fatal(parseSchemaErr)
	}

	return parsedSchema
}
