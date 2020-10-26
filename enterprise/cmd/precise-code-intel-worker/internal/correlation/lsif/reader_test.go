package lsif

import (
	"bytes"
	"context"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadLines(t *testing.T) {
	var content []byte
	for i := 0; i < 10000; i++ {
		content = append(content, strconv.Itoa(i)...)
		content = append(content, '\n')
	}

	unmarshal := func(line []byte) (Element, error) {
		id, err := strconv.Atoi(string(line))
		if err != nil {
			return Element{}, err
		}

		return Element{ID: id}, nil
	}

	pairs := readLines(context.Background(), bytes.NewReader(content), unmarshal)

	var ids []int
	for pair := range pairs {
		if pair.Err != nil {
			t.Fatalf("unexpected error: %s", pair.Err)
		}

		ids = append(ids, pair.Element.ID)
	}

	var expectedIDs []int
	for i := 0; i < 10000; i++ {
		expectedIDs = append(expectedIDs, i)
	}

	if diff := cmp.Diff(expectedIDs, ids); diff != "" {
		t.Errorf("unexpected ids (-want +got):\n%s", diff)
	}
}
