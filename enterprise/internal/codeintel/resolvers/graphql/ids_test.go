package graphql

import (
	"encoding/base64"
	"testing"

	"github.com/graph-gophers/graphql-go"
)

func TestUploadID(t *testing.T) {
	expected := int64(42)
	value, err := unmarshalLSIFUploadGQLID(marshalLSIFUploadGQLID(expected))
	if err != nil {
		t.Fatalf("unexpected error marshalling id: %s", err)
	}
	if value != expected {
		t.Errorf("unexpected id. have=%d want=%d", expected, value)
	}
}

func TestUnmarshalUploadIDString(t *testing.T) {
	expected := int64(42)
	id := graphql.ID(base64.StdEncoding.EncodeToString([]byte(`LSIFUpload:"42"`)))
	value, err := unmarshalLSIFUploadGQLID(id)
	if err != nil {
		t.Fatalf("unexpected error marshalling id: %s", err)
	}
	if value != expected {
		t.Errorf("unexpected id. have=%d want=%d", expected, value)
	}
}

func TestIndexID(t *testing.T) {
	expected := int64(42)
	value, err := unmarshalLSIFIndexGQLID(marshalLSIFIndexGQLID(expected))
	if err != nil {
		t.Fatalf("unexpected error marshalling id: %s", err)
	}
	if value != expected {
		t.Errorf("unexpected id. have=%d want=%d", expected, value)
	}
}
