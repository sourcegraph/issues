package graphqlbackend

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go/gqltesting"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/db"
)

func BenchmarkPrometheusFieldName(b *testing.B) {
	tests := [][3]string{
		{"Query", "settingsSubject", "settingsSubject"},
		{"SearchResultMatch", "highlights", "highlights"},
		{"TreeEntry", "isSingleChild", "isSingleChild"},
		{"NoMatch", "NotMatch", "other"},
	}
	for i, t := range tests {
		typeName, fieldName, want := t[0], t[1], t[2]
		b.Run(fmt.Sprintf("test-%v", i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				got := prometheusFieldName(typeName, fieldName)
				if got != want {
					b.Fatalf("got %q want %q", got, want)
				}
			}
		})
	}
}

func TestRepository(t *testing.T) {
	resetMocks()
	db.Mocks.Repos.MockGetByName(t, "github.com/gorilla/mux", 2)
	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: mustParseGraphQLSchema(t),
			Query: `
				{
					repository(name: "github.com/gorilla/mux") {
						name
					}
				}
			`,
			ExpectedResult: `
				{
					"repository": {
						"name": "github.com/gorilla/mux"
					}
				}
			`,
		},
	})
}

func TestResolverTo(t *testing.T) {
	// This test exists purely to remove some non determinism in our tests
	// run. The To* resolvers are stored in a map in our graphql
	// implementation => the order we call them is non deterministic =>
	// codecov coverage reports are noisy.
	resolvers := []interface{}{
		&FileMatchResolver{},
		&GitTreeEntryResolver{},
		&NamespaceResolver{},
		&NodeResolver{},
		&RepositoryResolver{},
		&CommitSearchResultResolver{},
		&gitRevSpec{},
		&searchSuggestionResolver{},
		&settingsSubject{},
		&statusMessageResolver{},
		&versionContextResolver{},
	}
	for _, r := range resolvers {
		typ := reflect.TypeOf(r)
		for i := 0; i < typ.NumMethod(); i++ {
			if name := typ.Method(i).Name; strings.HasPrefix(name, "To") {
				reflect.ValueOf(r).MethodByName(name).Call(nil)
			}
		}
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Verbose() {
		log15.Root().SetHandler(log15.LvlFilterHandler(log15.LvlError, log15.Root().GetHandler()))
		log.SetOutput(ioutil.Discard)
	}
	os.Exit(m.Run())
}
