package reposource

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
)

// RepoSource is a wrapper around a repository source (typically a code host config) that provides a
// method to map clone URLs to repo names using only the configuration (i.e., no network requests).
type RepoSource interface {
	// cloneURLToRepoName maps a Git clone URL (format documented here:
	// https://git-scm.com/docs/git-clone#_git_urls_a_id_urls_a) to the expected repo name for the
	// repository on the code host.  It does not actually check if the repository exists in the code
	// host. It merely does the mapping based on the rules set in the code host config.
	//
	// If the clone URL does not correspond to a repository that could exist on the code host, the
	// empty string is returned and err is nil. If there is an unrelated error, an error is
	// returned.
	CloneURLToRepoName(cloneURL string) (repoName api.RepoName, err error)
}

var nonSCPURLRegex = lazyregexp.New(`^(git\+)?(https?|ssh|rsync|file|git)://`)

// parseCloneURL parses a git clone URL into a URL struct. It supports the SCP-style git@host:path
// syntax that is common among code hosts.
func parseCloneURL(cloneURL string) (*url.URL, error) {
	if nonSCPURLRegex.MatchString(cloneURL) {
		return url.Parse(cloneURL)
	}

	// Support SCP-style syntax
	u, err := url.Parse("fake://" + strings.Replace(cloneURL, ":", "/", 1))
	if err != nil {
		return nil, err
	}
	u.Scheme = ""
	u.Path = strings.TrimPrefix(u.Path, "/")
	return u, nil
}

// hostname returns the hostname of a URL without www.
func hostname(url *url.URL) string {
	return strings.TrimPrefix(url.Hostname(), "www.")
}

// parseURLs parses the clone URL and repository host base URL into structs. It also returns a
// boolean indicating whether the hostnames of the URLs match.
func parseURLs(cloneURL, baseURL string) (parsedCloneURL, parsedBaseURL *url.URL, equalHosts bool, err error) {
	if baseURL != "" {
		parsedBaseURL, err = url.Parse(baseURL)
		if err != nil {
			return nil, nil, false, errors.Errorf("Error parsing baseURL: %s", err)
		}
		parsedBaseURL = extsvc.NormalizeBaseURL(parsedBaseURL)
	}

	parsedCloneURL, err = parseCloneURL(cloneURL)
	if err != nil {
		return nil, nil, false, errors.Errorf("Error parsing cloneURL: %s", err)
	}
	hostsMatch := parsedBaseURL != nil && hostname(parsedBaseURL) == hostname(parsedCloneURL)
	return parsedCloneURL, parsedBaseURL, hostsMatch, nil
}

type NameTransformationKind string

const (
	NameTransformationRegex NameTransformationKind = "regex"
)

// NameTransformation describes the rule to transform a repository name.
type NameTransformation struct {
	kind NameTransformationKind

	// Fields for regex replacement transformation.
	regexp      *regexp.Regexp
	replacement string

	// Note: Please add a blank line between each set of fields for a transformation rule
	// to help better organize the structure and more clear to the future contributors.
}

type NameTransformationOptions struct {
	// Options for regex replacement transformation.
	Regex       string
	Replacement string
}

func NewNameTransformation(opts NameTransformationOptions) (NameTransformation, error) {
	switch {
	case opts.Regex != "":
		r, err := regexp.Compile(opts.Regex)
		if err != nil {
			return NameTransformation{}, errors.Errorf("regexp.Compile %q: %v", opts.Regex, err)
		}
		return NameTransformation{
			kind:        NameTransformationRegex,
			regexp:      r,
			replacement: opts.Replacement,
		}, nil

	default:
		return NameTransformation{}, errors.Errorf("unrecognized transformation: %v", opts)
	}
}

func (nt NameTransformation) Kind() NameTransformationKind {
	return nt.kind
}

// Transform performs the transformation to given string.
func (nt NameTransformation) Transform(s string) string {
	switch nt.kind {
	case NameTransformationRegex:
		if nt.regexp != nil {
			s = nt.regexp.ReplaceAllString(s, nt.replacement)
		}
	}
	return s
}

// NameTransformations is a list of transformation rules.
type NameTransformations []NameTransformation

// Transform iterates and performs the list of transformations.
func (nts NameTransformations) Transform(s string) string {
	for _, nt := range nts {
		s = nt.Transform(s)
	}
	return s
}
