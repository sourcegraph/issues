package extsvc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtractRateLimitConfig(t *testing.T) {
	for _, tc := range []struct {
		name        string
		config      string
		kind        string
		displayName string
		want        RateLimitConfig
	}{
		{
			name:        "GitLab default",
			config:      `{"url": "https://example.com/"}`,
			kind:        KindGitLab,
			displayName: "GitLab 1",
			want: RateLimitConfig{
				BaseURL:     "https://example.com/",
				DisplayName: "GitLab 1",
				Limit:       10.0,
				IsDefault:   true,
			},
		},
		{
			name:        "GitHub default",
			config:      `{"url": "https://example.com/"}`,
			kind:        KindGitHub,
			displayName: "GitHub 1",
			want: RateLimitConfig{
				BaseURL:     "https://example.com/",
				DisplayName: "GitHub 1",
				Limit:       1.3888888888888888,
				IsDefault:   true,
			},
		},
		{
			name:        "Bitbucket Server default",
			config:      `{"url": "https://example.com/"}`,
			kind:        KindBitbucketServer,
			displayName: "BitbucketServer 1",
			want: RateLimitConfig{
				BaseURL:     "https://example.com/",
				DisplayName: "BitbucketServer 1",
				Limit:       8.0,
				IsDefault:   true,
			},
		},
		{
			name:        "Bitbucket Cloud default",
			config:      `{"url": "https://example.com/"}`,
			kind:        KindBitbucketCloud,
			displayName: "BitbucketCloud 1",
			want: RateLimitConfig{
				BaseURL:     "https://example.com/",
				DisplayName: "BitbucketCloud 1",
				Limit:       2.0,
				IsDefault:   true,
			},
		},
		{
			name:        "GitLab non-default",
			config:      `{"url": "https://example.com/", "rateLimit": {"enabled": true, "requestsPerHour": 3600}}`,
			kind:        KindGitLab,
			displayName: "GitLab 1",
			want: RateLimitConfig{
				BaseURL:     "https://example.com/",
				DisplayName: "GitLab 1",
				Limit:       1.0,
				IsDefault:   false,
			},
		},
		{
			name:        "GitHub default",
			config:      `{"url": "https://example.com/", "rateLimit": {"enabled": true, "requestsPerHour": 3600}}`,
			kind:        KindGitHub,
			displayName: "GitHub 1",
			want: RateLimitConfig{
				BaseURL:     "https://example.com/",
				DisplayName: "GitHub 1",
				Limit:       1.0,
				IsDefault:   false,
			},
		},
		{
			name:        "Bitbucket Server default",
			config:      `{"url": "https://example.com/", "rateLimit": {"enabled": true, "requestsPerHour": 3600}}`,
			kind:        KindBitbucketServer,
			displayName: "BitbucketServer 1",
			want: RateLimitConfig{
				BaseURL:     "https://example.com/",
				DisplayName: "BitbucketServer 1",
				Limit:       1.0,
				IsDefault:   false,
			},
		},
		{
			name:        "Bitbucket Cloud default",
			config:      `{"url": "https://example.com/", "rateLimit": {"enabled": true, "requestsPerHour": 3600}}`,
			kind:        KindBitbucketCloud,
			displayName: "BitbucketCloud 1",
			want: RateLimitConfig{
				BaseURL:     "https://example.com/",
				DisplayName: "BitbucketCloud 1",
				Limit:       1.0,
				IsDefault:   false,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rlc, err := ExtractRateLimitConfig(tc.config, tc.kind, tc.displayName)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, rlc); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
