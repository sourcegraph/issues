package main

import (
	"fmt"
	"strings"

	"github.com/prometheus/alertmanager/api/v2/models"
)

func stringP(v string) *string {
	return &v
}

func boolP(v bool) *bool {
	return &v
}

const (
	matcherRegexPrefix = "^("
	matcherRegexSuffix = ")$"
)

// newMatchersFromSilence creates Alertmanager alert matchers from a configured silence
func newMatchersFromSilence(silence string) models.Matchers {
	return models.Matchers{{
		Name:    stringP("alertname"),
		Value:   stringP(fmt.Sprintf("%s%s%s", matcherRegexPrefix, silence, matcherRegexSuffix)),
		IsRegex: boolP(true),
	}}
}

// newSilenceFromMatchers returns the silenced alert from Alertmanager alert matchers
func newSilenceFromMatchers(matchers models.Matchers) string {
	for _, m := range matchers {
		if *m.Name == "alertname" {
			return strings.TrimSuffix(strings.TrimPrefix(*m.Value, matcherRegexPrefix), matcherRegexSuffix)
		}
	}
	return ""
}
