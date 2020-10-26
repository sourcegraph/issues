package db

import (
	"encoding/json"
	"testing"
)

func assertJSONEqual(t *testing.T, want, got interface{}) {
	want_j := asJSON(t, want)
	got_j := asJSON(t, got)
	if want_j != got_j {
		t.Errorf("Wanted %s, but got %s", want_j, got_j)
	}
}

func jsonEqual(t *testing.T, a, b interface{}) bool {
	return asJSON(t, a) == asJSON(t, b)
}

func asJSON(t *testing.T, v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
