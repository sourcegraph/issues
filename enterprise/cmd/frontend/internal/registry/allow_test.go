package registry

import (
	"reflect"
	"sort"
	"testing"
)

func sameElements(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))

	copy(aCopy, a)
	copy(bCopy, b)

	sort.Strings(aCopy)
	sort.Strings(bCopy)

	return reflect.DeepEqual(aCopy, bCopy)
}

func TestFilterRemoteExtensions(t *testing.T) {
	// defer licensing.TestingSkipFeatureChecks()()

	// run := func(allowRemoteExtensions *[]string, extensions []string, want []string) {
	// 	t.Helper()
	// 	if allowRemoteExtensions != nil {
	// 		conf.Mock(&conf.Unified{SiteConfiguration: schema.SiteConfiguration{Extensions: &schema.Extensions{AllowRemoteExtensions: *allowRemoteExtensions}}})
	// 		defer conf.Mock(nil)
	// 	}
	// 	var xs []*types.Extension
	// 	for _, id := range extensions {
	// 		xs = append(xs, &types.Extension{ExtensionID: id})
	// 	}
	// 	got := []string{}
	// 	for _, x := range FilterRemoteExtensions(xs) {
	// 		got = append(got, x.ExtensionID)
	// 	}
	// 	if !sameElements(got, want) {
	// 		t.Errorf("want %+v got %+v", want, got)
	// 	}
	// }

	// run(nil, []string{}, []string{})
	// run(nil, []string{"a"}, []string{"a"})
	// run(&[]string{}, []string{}, []string{})
	// run(&[]string{"a"}, []string{}, []string{})
	// run(&[]string{}, []string{"a"}, []string{})
	// run(&[]string{"a"}, []string{"b"}, []string{})
	// run(&[]string{"a"}, []string{"a"}, []string{"a"})
	// run(&[]string{"b", "c"}, []string{"a", "b", "c"}, []string{"b", "c"})
}
