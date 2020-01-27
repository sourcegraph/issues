package testutil

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func AssertGolden(t testing.TB, path string, update bool, want interface{}) {
	t.Helper()

	data := marshal(t, want)

	if update {
		if err := ioutil.WriteFile(path, data, 0640); err != nil {
			t.Fatalf("failed to update golden file %q: %s", path, err)
		}
	}

	golden, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file %q: %s", path, err)
	}

	if have, want := string(data), string(golden); have != want {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(have, want, false)
		t.Error(dmp.DiffPrettyText(diffs))
	}
}

func marshal(t testing.TB, v interface{}) []byte {
	t.Helper()

	switch v2 := v.(type) {
	case string:
		return []byte(v2)
	case []byte:
		return v2
	default:
		data, err := json.MarshalIndent(v, " ", " ")
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
}
