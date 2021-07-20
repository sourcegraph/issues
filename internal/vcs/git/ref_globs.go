package git

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/gobwas/glob"
)

// RefGlob describes a glob pattern that either includes or excludes refs. Exactly 1 of the fields
// must be set.
type RefGlob struct {
	// Include is a glob pattern for including refs interpreted as in `git log --glob`. See the
	// git-log(1) manual page for details.
	Include string

	// Exclude is a glob pattern for excluding refs interpreted as in `git log --exclude`. See the
	// git-log(1) manual page for details.
	Exclude string
}

// RefGlobs is a compiled matcher based on RefGlob patterns. Use CompileRefGlobs to create it.
type RefGlobs []compiledRefGlobPattern

type compiledRefGlobPattern struct {
	pattern glob.Glob
	include bool // true for include, false for exclude
}

// CompileRefGlobs compiles the ordered ref glob patterns (interpreted as in `git log --glob
// ... --exclude ...`; see the git-log(1) manual page) into a matcher. If the input patterns are
// invalid, an error is returned.
func CompileRefGlobs(globs []RefGlob) (RefGlobs, error) {
	c := make(RefGlobs, len(globs))
	for i, g := range globs {
		// Validate exclude globs according to `git log --exclude`'s specs: "The patterns
		// given...must begin with refs/... If a trailing /* is intended, it must be given
		// explicitly."
		if g.Exclude != "" {
			if !strings.HasPrefix(g.Exclude, "refs/") {
				return nil, errors.Errorf(`git ref exclude glob must begin with "refs/" (got %q)`, g.Exclude)
			}
		}

		// Add implicits (according to `git log --glob`'s specs).
		if g.Include != "" {
			// `git log --glob`: "Leading refs/ is automatically prepended if missing.".
			if !strings.HasPrefix(g.Include, "refs/") {
				g.Include = "refs/" + g.Include
			}

			// `git log --glob`: "If pattern lacks ?, *, or [, /* at the end is implied." Also
			// support an important undocumented case: support exact matches. For example, the
			// pattern refs/heads/a should match the ref refs/heads/a (i.e., just appending /* to
			// the pattern would yield refs/heads/a/*, which would *not* match refs/heads/a, so we
			// need to make the /* optional).
			if !strings.ContainsAny(g.Include, "?*[") {
				var suffix string
				if strings.HasSuffix(g.Include, "/") {
					suffix = "*"
				} else {
					suffix = "/*"
				}
				g.Include += "{," + suffix + "}"
			}
		}

		var pattern string
		if g.Include != "" {
			pattern = g.Include
			c[i].include = true
		} else {
			pattern = g.Exclude
		}
		var err error
		c[i].pattern, err = glob.Compile(pattern)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

// Match reports whether the named ref matches the ref globs.
func (gs RefGlobs) Match(ref string) bool {
	match := false
	for _, g := range gs {
		if g.include == match {
			// If the glob does not change the outcome, skip it. (For example, if the ref is already
			// matched, and the next glob is another include glob.)
			continue
		}
		if g.pattern.Match(ref) {
			match = g.include
		}
	}
	return match
}
