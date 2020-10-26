package indexer

import (
	"fmt"
	"testing"
)

func TestSanitizeImage(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			// no regex match (no crash)
			input:    "",
			expected: "",
		},
		{
			// no tag or hash
			input:    "sourcegraph/ignite-ubuntu",
			expected: "sourcegraph/ignite-ubuntu",
		},
		{
			// tag only
			input:    "sourcegraph/ignite-ubuntu:insiders",
			expected: "sourcegraph/ignite-ubuntu:insiders",
		},
		{
			// remove hash without tag
			input:    "sourcegraph/ignite-ubuntu@sha256:e54a802a8bec44492deee944acc560e4e0a98f6ffa9a5038f0abac1af677e134",
			expected: "sourcegraph/ignite-ubuntu",
		},
		{
			// tag and hash - keep only tag
			input:    "sourcegraph/ignite-ubuntu:insiders@sha256:e54a802a8bec44492deee944acc560e4e0a98f6ffa9a5038f0abac1af677e134",
			expected: "sourcegraph/ignite-ubuntu:insiders",
		},
	}

	for _, testCase := range testCases {
		name := fmt.Sprintf("input=%s", testCase.input)

		t.Run(name, func(t *testing.T) {
			if image := sanitizeImage(testCase.input); image != testCase.expected {
				t.Errorf("unexpected image. want=%q have=%q", testCase.expected, image)
			}
		})
	}
}
