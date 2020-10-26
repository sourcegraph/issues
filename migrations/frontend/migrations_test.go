package migrations_test

import (
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	migrations "github.com/sourcegraph/sourcegraph/migrations/frontend"
)

const FirstMigration = 1528395684

func TestIDConstraints(t *testing.T) {
	ups, err := filepath.Glob("*.up.sql")
	if err != nil {
		t.Fatal(err)
	}

	byID := map[int][]string{}
	for _, name := range ups {
		id, err := strconv.Atoi(name[:strings.IndexByte(name, '_')])
		if err != nil {
			t.Fatalf("failed to parse name %q: %v", name, err)
		}
		byID[id] = append(byID[id], name)
	}

	for id, names := range byID {
		// Check if we are using sequential migrations from a certain point.
		if _, hasPrev := byID[id-1]; id > FirstMigration && !hasPrev {
			t.Errorf("migration with ID %d exists, but previous one (%d) does not", id, id-1)
		}
		if len(names) > 1 {
			t.Errorf("multiple migrations with ID %d: %s", id, strings.Join(names, " "))
		}
	}
}

func TestNeedsGenerate(t *testing.T) {
	want, err := filepath.Glob("*.sql")
	if err != nil {
		t.Fatal(err)
	}
	got := migrations.AssetNames()
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatal("bindata out of date. Please run:\n  go generate github.com/sourcegraph/sourcegraph/migrations/...")
	}
}
