package httpapi

import (
	"compress/gzip"
	"errors"
	"net/http"
	"strings"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

func serveGraphQL(schema *graphql.Schema) func(w http.ResponseWriter, r *http.Request) (err error) {
	relayHandler := &relay.Handler{Schema: schema}
	return func(w http.ResponseWriter, r *http.Request) (err error) {
		if r.Method != "POST" {
			// The URL router should not have routed to this handler if method is not POST, but just in
			// case.
			return errors.New("method must be POST")
		}

		// We use the query to denote the name of a GraphQL request, e.g. for /.api/graphql?Repositories
		// the name is "Repositories".
		requestName := "unknown"
		if r.URL.RawQuery != "" {
			requestName = r.URL.RawQuery
		}
		r = r.WithContext(trace.WithGraphQLRequestName(r.Context(), requestName))

		r = r.WithContext(trace.WithRequestSource(r.Context(), guessSource(r)))

		if r.Header.Get("Content-Encoding") == "gzip" {
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				return err
			}

			r.Body = gzipReader

			defer gzipReader.Close()
		}

		relayHandler.ServeHTTP(w, r)
		return nil
	}
}

// guessSource guesses the source the request came from (browser, other HTTP client, etc.)
func guessSource(r *http.Request) trace.SourceType {
	userAgent := r.UserAgent()
	for _, guess := range []string{
		"Mozilla",
		"WebKit",
		"Gecko",
		"Chrome",
		"Firefox",
		"Safari",
		"Edge",
	} {
		if strings.Contains(userAgent, guess) {
			return trace.SourceBrowser
		}
	}
	return trace.SourceOther
}
