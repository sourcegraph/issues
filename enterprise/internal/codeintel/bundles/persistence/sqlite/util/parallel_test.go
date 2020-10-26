package util

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInvokeAllBlocks(t *testing.T) {
	vs := make(chan int, 3)
	err := InvokeAll(
		func() error { vs <- 1; return nil },
		func() error { vs <- 2; return nil },
		func() error { vs <- 3; return nil },
	)
	close(vs)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	var values []int
	for v := range vs {
		values = append(values, v)
	}
	sort.Ints(values)

	if diff := cmp.Diff([]int{1, 2, 3}, values); diff != "" {
		t.Fatalf("unexpected channel result (-want +got):\n%s", diff)
	}
}

func TestInvokeAllMultipleErrors(t *testing.T) {
	err := InvokeAll(
		func() error { return fmt.Errorf("err1") },
		func() error { return fmt.Errorf("err2") },
		func() error { return fmt.Errorf("err3") },
	)
	if err == nil {
		t.Fatalf("unexpected nil error")
	}

	for _, sub := range []string{"err1", "err2", "err3"} {
		if !strings.Contains(err.Error(), sub) {
			t.Errorf("expected error to contain %s: %s", sub, err)
		}
	}
}

func TestInvokeN(t *testing.T) {
	vs := make(chan int, 12)
	err := InvokeN(12, func() error { vs <- 1; return nil })
	close(vs)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	n := 0
	for range vs {
		n++
	}

	if n != 12 {
		t.Fatalf("expected %d values, got %d", 12, n)
	}
}
