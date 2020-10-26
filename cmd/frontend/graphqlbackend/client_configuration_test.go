package graphqlbackend

import "testing"

func TestStripPassword(t *testing.T) {
	tests := []struct {
		u    string
		want string
	}{
		{
			u:    "http://example.com/",
			want: "http://example.com/",
		},
		{
			u:    "a string",
			want: "a string",
		},
		{
			u:    "http://user:pass@example.com/",
			want: "http://user:***@example.com/",
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			have := stripPassword(test.u)
			if have != test.want {
				t.Fatalf("Have %q, want %q", have, test.want)
			}
		})
	}
}
