package authz

import (
	"reflect"
	"testing"
)

func TestParseAuthorizationHeader(t *testing.T) {
	tests := map[string]struct {
		token    string
		sudoUser string
		err      bool
	}{
		"token tok":                              {token: "tok"},
		"token tok==":                            {token: "tok=="},
		`token token=tok`:                        {token: "tok"},
		`token token="tok=="`:                    {token: "tok=="},
		`token-sudo token="tok==", user="alice"`: {token: "tok==", sudoUser: "alice"},
		`token-sudo token=tok, user="alice"`:     {token: "tok", sudoUser: "alice"},
		`token-sudo token="tok==", user=alice`:   {token: "tok==", sudoUser: "alice"},
		"xyz tok":                                {err: true},
		`token-sudo user="alice"`:                {err: true},
		`token-sudo token="",user="alice"`:       {err: true},
		`token k=v, k=v`:                         {err: true},
	}
	for input, test := range tests {
		t.Run(input, func(t *testing.T) {
			token, sudoUser, err := ParseAuthorizationHeader(input)
			if (err != nil) != test.err {
				t.Errorf("got error %v, want error? %v", err, test.err)
			}
			if err != nil {
				return
			}
			if token != test.token {
				t.Errorf("got token %q, want %q", token, test.token)
			}
			if sudoUser != test.sudoUser {
				t.Errorf("got sudoUser %+v, want %+v", sudoUser, test.sudoUser)
			}
		})
	}
}

func TestParseHTTPCredentials(t *testing.T) {
	tests := map[string]struct {
		scheme  string
		token68 string
		params  map[string]string
		err     bool
	}{
		"scheme v1":                 {scheme: "scheme", token68: "v1"},
		"scheme v1==":               {scheme: "scheme", token68: "v1=="},
		`scheme k1="v1"`:            {scheme: "scheme", params: map[string]string{"k1": "v1"}},
		`scheme-2 k1="v1", k2="v2"`: {scheme: "scheme-2", params: map[string]string{"k1": "v1", "k2": "v2"}},
		`scheme-2 k1=v1, k2="v2"`:   {scheme: "scheme-2", params: map[string]string{"k1": "v1", "k2": "v2"}},
		`scheme k=v, k=v`:           {err: true},
	}
	for input, test := range tests {
		t.Run(input, func(t *testing.T) {
			scheme, token68, params, err := parseHTTPCredentials(input)
			if (err != nil) != test.err {
				t.Errorf("got error %v, want error? %v", err, test.err)
			}
			if err != nil {
				return
			}
			if scheme != test.scheme {
				t.Errorf("got scheme %q, want %q", scheme, test.scheme)
			}
			if token68 != test.token68 {
				t.Errorf("got token68 %q, want %q", token68, test.token68)
			}
			if !reflect.DeepEqual(params, test.params) {
				t.Errorf("got params %+v, want %+v", params, test.params)
			}
		})
	}
}
