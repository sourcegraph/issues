package conf

import (
	"reflect"
	"testing"

	"github.com/sourcegraph/sourcegraph/schema"
)

func TestAuthPublic(t *testing.T) {
	tests := map[string]struct {
		input Unified
		want  bool
	}{
		"false": {
			input: Unified{SiteConfiguration: schema.SiteConfiguration{AuthPublic: false}},
			want:  false,
		},
		"true, no auth provider": {
			input: Unified{SiteConfiguration: schema.SiteConfiguration{AuthPublic: true}},
			want:  true,
		},
		"true, non-builtin auth provider": {
			input: Unified{SiteConfiguration: schema.SiteConfiguration{AuthPublic: true, AuthProviders: []schema.AuthProviders{{Openidconnect: &schema.OpenIDConnectAuthProvider{}}}}},
			want:  true,
		},
		"true, builtin auth provider": {
			input: Unified{SiteConfiguration: schema.SiteConfiguration{AuthPublic: true, AuthProviders: []schema.AuthProviders{{Builtin: &schema.BuiltinAuthProvider{}}}}},
			want:  true,
		},
		"false, builtin auth provider": {
			input: Unified{SiteConfiguration: schema.SiteConfiguration{AuthPublic: false, AuthProviders: []schema.AuthProviders{{Builtin: &schema.BuiltinAuthProvider{}}}}},
			want:  false,
		},
	}
	for label, test := range tests {
		t.Run(label, func(t *testing.T) {
			got := authPublic(&test.input)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}
