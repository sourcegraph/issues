package db

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

func TestErrorsInterface(t *testing.T) {
	cases := []struct {
		Err       error
		Predicate func(error) bool
	}{
		{&RepoNotFoundErr{}, errcode.IsNotFound},
		{userNotFoundErr{}, errcode.IsNotFound},
	}
	for _, c := range cases {
		if !c.Predicate(c.Err) {
			t.Errorf("%s does not match predicate %s", c.Err.Error(), functionName(c.Predicate))
		}
	}
}

func functionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
