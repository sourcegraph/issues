package streaming

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFilters(t *testing.T) {
	// Add lots of repos, files and fork. Ensure we compute a good summary
	// which balances types.

	m := make(filters)
	for count := int32(1); count <= 1000; count++ {
		repo := fmt.Sprintf("repo-%d", count)
		m.Add(repo, repo, count, false, "repo")
	}
	for count := int32(1); count <= 100; count++ {
		file := fmt.Sprintf("file-%d", count)
		m.Add(file, file, count, false, "file")
	}
	// Add one large file count to see if it is recommended near the top.
	m.Add("file-big", "file-big", 10000, false, "file")

	// Test important and updating
	m.Add("fork", "fork", 3, false, "repo")
	m.MarkImportant("fork")
	m.Add("fork", "fork", 1, false, "repo")

	want := []string{
		"fork 4",
		"file-big 10000",
		"repo-1000 1000",
		"repo-999 999",
		"repo-998 998",
		"repo-997 997",
		"repo-996 996",
		"repo-995 995",
		"repo-994 994",
		"repo-993 993",
		"repo-992 992",
		"repo-991 991",
		"repo-990 990",
		"file-100 100",
		"file-99 99",
		"file-98 98",
		"file-97 97",
		"file-96 96",
		"file-95 95",
		"file-94 94",
		"file-93 93",
		"file-92 92",
		"file-91 91",
		"file-90 90",
	}

	filters := m.Compute(computeOpts{
		MaxRepos: 12,
		MaxOther: 12,
	})
	var got []string
	for _, f := range filters {
		got = append(got, fmt.Sprintf("%s %d", f.Value, f.Count))
	}

	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("mismatch (-want +got):\n%s", d)
	}
}
