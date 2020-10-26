package main

import "strings"

func contains(haystack []string, needle string) bool {
	for _, candidate := range haystack {
		if candidate == needle {
			return true
		}
	}

	return false
}

func nonTrackingLabels(labels []string) (filtered []string) {
	for _, label := range labels {
		if label != "tracking" {
			filtered = append(filtered, label)
		}
	}

	return filtered
}

func redactLabels(labels []string) (redacted []string) {
	for _, label := range labels {
		if strings.HasPrefix(label, "estimate/") || strings.HasPrefix(label, "planned/") {
			redacted = append(redacted, label)
		}
	}

	return redacted
}
