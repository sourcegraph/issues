/*
 * Copyright (c) 2014 Will Maier <wcmaier@m.aier.us>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package vcs

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
)

// ParseURL parses rawurl into a URL structure. Parse first attempts to
// find a standard URL with a valid VCS URL scheme. If
// that cannot be found, it then attempts to find a SCP-like URL.
// And if that cannot be found, it assumes rawurl is a local path.
// If none of these rules apply, Parse returns an error.
//
// Code copied and modified from github.com/whilp/git-urls to support perforce scheme.
func ParseURL(rawurl string) (u *URL, err error) {
	parsers := []func(string) (*URL, error){
		parseScheme,
		parseScp,
		parseLocal,
	}

	// Apply each parser in turn; if the parser succeeds, accept its
	// result and return.
	for _, p := range parsers {
		u, err = p(rawurl)
		if err == nil {
			return u, err
		}
	}

	// It's unlikely that none of the parsers will succeed, since
	// ParseLocal is very forgiving.
	return new(URL), errors.Errorf("failed to parse %q", rawurl)
}

var schemes = map[string]struct{}{
	"ssh":      {},
	"git":      {},
	"git+ssh":  {},
	"http":     {},
	"https":    {},
	"ftp":      {},
	"ftps":     {},
	"rsync":    {},
	"file":     {},
	"perforce": {},
}

func parseScheme(rawurl string) (*URL, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	if _, valid := schemes[u.Scheme]; !valid {
		return nil, errors.Errorf("scheme %q is not a valid transport", u.Scheme)
	}

	return &URL{format: formatStdlib, URL: *u}, nil
}

const (
	// usernameRe is the regexp for the username part in a repo URL. Eg: sourcegraph@
	usernameRe = "([a-zA-Z0-9-._~]+@)"

	// urlRe is the regexp for the url part in a repo URL. Eg: github.com
	urlRe = "([a-zA-Z0-9._-]+)"

	// repoRe is the regexp for the repo in a repo URL. Eg: sourcegraph/sourcegraph
	repoRe = `([a-zA-Z0-9\@./._-]+)(?:\?||$)(.*)`
)

// scpSyntax was modified from https://golang.org/src/cmd/go/vcs.go.
var scpSyntax = regexp.MustCompile(fmt.Sprintf(`^%s?%s:%s$`, usernameRe, urlRe, repoRe))

// parseScp parses rawurl into a URL object. The rawurl must be
// an SCP-like URL, otherwise ParseScp returns an error.
func parseScp(rawurl string) (*URL, error) {
	match := scpSyntax.FindAllStringSubmatch(rawurl, -1)
	if len(match) == 0 {
		return nil, errors.Errorf("no scp URL found in %q", rawurl)
	}
	m := match[0]
	user := strings.TrimRight(m[1], "@")
	var userinfo *url.Userinfo
	if user != "" {
		userinfo = url.User(user)
	}
	rawquery := ""
	if len(m) > 3 {
		rawquery = m[4]
	}
	return &URL{
		format: formatRsync,
		URL: url.URL{
			User:     userinfo,
			Host:     m[2],
			Path:     m[3],
			RawQuery: rawquery,
		},
	}, nil
}

// parseLocal parses rawurl into a URL object with a "file"
// scheme. This will effectively never return an error.
func parseLocal(rawurl string) (*URL, error) {
	return &URL{
		format: formatLocal,
		URL: url.URL{
			Scheme: "file",
			Host:   "",
			Path:   rawurl,
		},
	}, nil
}

// URL wraps url.URL to provide rsync format compatible `String()` functionality.
// eg git@foo.com:foo/bar.git
// stdlib URL.String() would marshal those URLs with a leading slash in the path, which for
// standard git hosts changes path semantics. This function will only use stdlib URL.String()
// if a scheme is specified, otherwise it uses a custom format built for compatibility
type URL struct {
	url.URL

	format urlFormat
}

type urlFormat int

const (
	formatStdlib urlFormat = iota
	formatRsync
	formatLocal
)

// String will return standard url.URL.String() if the url has a .Scheme set, but if
// not it will produce an rsync format URL, eg `git@foo.com:foo/bar.git`
func (u *URL) String() string {
	// only use custom String() implementation for rsync format URLs
	if u.format != formatRsync {
		return u.URL.String()
	}
	// otherwise attempt to marshal scp style URLs
	var buf strings.Builder
	if u.User != nil {
		buf.WriteString(u.User.String())
		buf.WriteByte('@')
	}
	if h := u.Host; h != "" {
		buf.WriteString(h)
		// key difference here, add : and don't add a leading / to the path
		buf.WriteByte(':')
	}
	buf.WriteString(u.EscapedPath())
	if u.RawQuery != "" {
		buf.WriteByte('?')
		buf.WriteString(u.RawQuery)
	}
	if u.Fragment != "" {
		buf.WriteByte('#')
		buf.WriteString(u.EscapedFragment())
	}
	return buf.String()
}

// IsSSH returns whether this URL is SSH based, which for vcs.URL means
// if the scheme is either empty or `ssh`, this is because of rsync format
// urls being cloned over SSH, but not including a scheme.
func (u *URL) IsSSH() bool {
	return u.Scheme == "ssh" || u.Scheme == ""
}
