package graphqlbackend

import "testing"

func TestGitCommitBody(t *testing.T) {
	tests := map[string]string{
		"hello":               "",
		"hello\n":             "",
		"hello\n\n":           "",
		"hello\nworld":        "world",
		"hello\n\nworld":      "world",
		"hello\n\nworld\nfoo": "world\nfoo",
	}
	for input, want := range tests {
		got := GitCommitBody(input)
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
