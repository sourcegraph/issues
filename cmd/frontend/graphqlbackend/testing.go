package graphqlbackend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/google/go-cmp/cmp"
	"github.com/graph-gophers/graphql-go"
	gqlerrors "github.com/graph-gophers/graphql-go/errors"
)

var (
	parseSchemaOnce sync.Once
	parseSchemaErr  error
	parsedSchema    *graphql.Schema
)

func mustParseGraphQLSchema(t *testing.T) *graphql.Schema {
	t.Helper()

	parseSchemaOnce.Do(func() {
		parsedSchema, parseSchemaErr = NewSchema(nil, nil, nil, nil, nil, nil, nil, nil)
	})
	if parseSchemaErr != nil {
		t.Fatal(parseSchemaErr)
	}

	return parsedSchema
}

// Code below copied from graph-gophers and has been modified to improve
// error messages

// Test is a GraphQL test case to be used with RunTest(s).
type Test struct {
	Context        context.Context
	Schema         *graphql.Schema
	Query          string
	OperationName  string
	Variables      map[string]interface{}
	ExpectedResult string
	ExpectedErrors []*gqlerrors.QueryError
}

// RunTests runs the given GraphQL test cases as subtests.
func RunTests(t *testing.T, tests []*Test) {
	if len(tests) == 1 {
		RunTest(t, tests[0])
		return
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			RunTest(t, test)
		})
	}
}

// RunTest runs a single GraphQL test case.
func RunTest(t *testing.T, test *Test) {
	t.Helper()

	if test.Context == nil {
		test.Context = context.Background()
	}
	result := test.Schema.Exec(test.Context, test.Query, test.OperationName, test.Variables)

	checkErrors(t, test.ExpectedErrors, result.Errors)

	if test.ExpectedResult == "" {
		if result.Data != nil {
			t.Errorf("got: %s", result.Data)
			t.Fatal("want: null")
		}
		return
	}

	// Verify JSON to avoid red herring errors.
	got, err := formatJSON(result.Data)
	if err != nil {
		t.Fatalf("got: invalid JSON: %s", err)
	}
	want, err := formatJSON([]byte(test.ExpectedResult))
	if err != nil {
		t.Fatalf("want: invalid JSON: %s", err)
	}

	if !bytes.Equal(got, want) {
		t.Logf("got:  %s", got)
		t.Logf("want: %s", want)
		t.Fail()
	}
}

func formatJSON(data []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	formatted, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return formatted, nil
}

func checkErrors(t *testing.T, want, got []*gqlerrors.QueryError) {
	t.Helper()

	sortErrors(want)
	sortErrors(got)

	// Compare without caring about the concrete type of the error returned
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(gqlerrors.QueryError{}, "ResolverError")); diff != "" {
		t.Fatal(diff)
	}
}

func sortErrors(errs []*gqlerrors.QueryError) {
	if len(errs) <= 1 {
		return
	}
	sort.Slice(errs, func(i, j int) bool {
		return fmt.Sprintf("%s", errs[i].Path) < fmt.Sprintf("%s", errs[j].Path)
	})
}
