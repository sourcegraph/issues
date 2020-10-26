package main

import (
	"testing"
)

func TestMatchersAndSilences(t *testing.T) {
	tests := []struct {
		name                  string
		silence               string
		wantMatcherAlertnames []string
	}{
		{
			name:                  "add strict match",
			silence:               "hello",
			wantMatcherAlertnames: []string{"^(hello)$"},
		},
		{
			name:                  "accept regex",
			silence:               ".*hello.*",
			wantMatcherAlertnames: []string{"^(.*hello.*)$"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matchers := newMatchersFromSilence(tt.silence)
			for i, m := range matchers {
				if *m.Name == "alertname" {
					if *m.Value != tt.wantMatcherAlertnames[i] {
						t.Errorf("newMatchersFromSilence got %s, want %s",
							*m.Value, tt.wantMatcherAlertnames[i])
					}
				}
			}
			silence := newSilenceFromMatchers(matchers)
			if silence != tt.silence {
				t.Errorf("newSilenceFromMatchers() = %v, want %v", silence, tt.silence)
			}
		})
	}
}
