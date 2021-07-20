package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/search/streaming/api"
	streamhttp "github.com/sourcegraph/sourcegraph/internal/search/streaming/http"
)

type streamClient struct {
	token    string
	endpoint string
	client   *http.Client
}

func newStreamClient() (*streamClient, error) {
	tkn := os.Getenv(envToken)
	if tkn == "" {
		return nil, errors.Errorf("%s not set", envToken)
	}
	endpoint := os.Getenv(envEndpoint)
	if endpoint == "" {
		return nil, errors.Errorf("%s not set", envEndpoint)
	}

	return &streamClient{
		token:    tkn,
		endpoint: endpoint,
		client:   http.DefaultClient,
	}, nil
}

func (s *streamClient) search(ctx context.Context, query, queryName string) (*metrics, error) {
	req, err := streamhttp.NewRequest(s.endpoint, query)
	if err != nil {
		return nil, errors.Errorf("create request: %w", err)
	}
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "token "+s.token)
	req.Header.Set("X-Sourcegraph-Should-Trace", "true")
	req.Header.Set("User-Agent", fmt.Sprintf("SearchBlitz (%s)", queryName))

	var m metrics
	first := true

	start := time.Now()

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	dec := streamhttp.Decoder{
		OnMatches: func(matches []streamhttp.EventMatch) {
			if first && len(matches) > 0 {
				m.firstResult = time.Since(start)
				first = false
			}
		},
		OnProgress: func(p *api.Progress) {
			m.matchCount = p.MatchCount
		},
	}

	if err := dec.ReadAll(resp.Body); err != nil {
		return nil, err
	}

	m.took = time.Since(start)
	m.trace = resp.Header.Get("x-trace")

	// If we have no results, we use the total time taken for first result
	// time.
	if first {
		m.firstResult = m.took
	}

	return &m, nil
}

func (s *streamClient) clientType() string {
	return "stream"
}
