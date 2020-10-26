package trace

import (
	"testing"

	"github.com/opentracing/opentracing-go"
)

func TestSpanURL(t *testing.T) {
	want := func(want string) {
		t.Helper()
		got := SpanURL(nil)
		if got != want {
			t.Fatalf("want %s, got %s", want, got)
		}
	}
	want("#tracer-not-enabled")
	SetSpanURLFunc(func(span opentracing.Span) string { return "test" })
	want("test")
	SetSpanURLFunc(nil)
	want("#tracer-not-enabled")
}
