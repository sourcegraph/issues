package search

import (
	"bytes"
	"context"
	"errors"
	"io"
	"regexp"
	"regexp/syntax"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/sourcegraph/sourcegraph/cmd/searcher/protocol"
	"github.com/sourcegraph/sourcegraph/internal/pathmatch"
	"github.com/sourcegraph/sourcegraph/internal/store"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"

	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
)

// readerGrep is responsible for finding LineMatches. It is not concurrency
// safe (it reuses buffers for performance).
//
// This code is base on reading the techniques detailed in
// http://blog.burntsushi.net/ripgrep/
//
// The stdlib regexp is pretty powerful and in fact implements many of the
// features in ripgrep. Our implementation gives high performance via pruning
// aggressively which files to consider (non-binary under a limit) and
// optimizing for assuming most lines will not contain a match. The pruning of
// files is done by the store.
//
// If there is no more low-hanging fruit and perf is not acceptable, we could
// consider using ripgrep directly (modify it to search zip archives).
//
// TODO(keegan) return search statistics
type readerGrep struct {
	// re is the regexp to match, or nil if empty ("match all files' content").
	re *regexp.Regexp

	// ignoreCase if true means we need to do case insensitive matching.
	ignoreCase bool

	// transformBuf is reused between file searches to avoid
	// re-allocating. It is only used if we need to transform the input
	// before matching. For example we lower case the input in the case of
	// ignoreCase.
	transformBuf []byte

	// matchPath is compiled from the include/exclude path patterns and reports
	// whether a file path matches (and should be searched).
	matchPath pathmatch.PathMatcher

	// literalSubstring is used to test if a file is worth considering for
	// matches. literalSubstring is guaranteed to appear in any match found by
	// re. It is the output of the longestLiteral function. It is only set if
	// the regex has an empty LiteralPrefix.
	literalSubstring []byte
}

// compile returns a readerGrep for matching p.
func compile(p *protocol.PatternInfo) (*readerGrep, error) {
	var (
		re               *regexp.Regexp
		literalSubstring []byte
	)
	if p.Pattern != "" {
		expr := p.Pattern
		if !p.IsRegExp {
			expr = regexp.QuoteMeta(expr)
		}
		if p.IsWordMatch {
			expr = `\b` + expr + `\b`
		}
		if p.IsRegExp {
			// We don't do the search line by line, therefore we want the
			// regex engine to consider newlines for anchors (^$).
			expr = "(?m:" + expr + ")"
		}
		if !p.IsCaseSensitive {
			// We don't just use (?i) because regexp library doesn't seem
			// to contain good optimizations for case insensitive
			// search. Instead we lowercase the input and pattern.
			re, err := syntax.Parse(expr, syntax.Perl)
			if err != nil {
				return nil, err
			}
			lowerRegexpASCII(re)
			expr = re.String()
		}

		var err error
		re, err = regexp.Compile(expr)
		if err != nil {
			return nil, err
		}

		// Only use literalSubstring optimization if the regex engine doesn't
		// have a prefix to use.
		if pre, _ := re.LiteralPrefix(); pre == "" {
			ast, err := syntax.Parse(expr, syntax.Perl)
			if err != nil {
				return nil, err
			}
			ast = ast.Simplify()
			literalSubstring = []byte(longestLiteral(ast))
		}
	}

	pathOptions := pathmatch.CompileOptions{
		RegExp:        p.PathPatternsAreRegExps,
		CaseSensitive: p.PathPatternsAreCaseSensitive,
	}
	matchPath, err := pathmatch.CompilePathPatterns(p.IncludePatterns, p.ExcludePattern, pathOptions)
	if err != nil {
		return nil, err
	}

	return &readerGrep{
		re:               re,
		ignoreCase:       !p.IsCaseSensitive,
		matchPath:        matchPath,
		literalSubstring: literalSubstring,
	}, nil
}

// Copy returns a copied version of rg that is safe to use from another
// goroutine.
func (rg *readerGrep) Copy() *readerGrep {
	return &readerGrep{
		re:               rg.re,
		ignoreCase:       rg.ignoreCase,
		matchPath:        rg.matchPath,
		literalSubstring: rg.literalSubstring,
	}
}

// matchString returns whether rg's regexp pattern matches s. It is intended to be
// used to match file paths.
func (rg *readerGrep) matchString(s string) bool {
	if rg.re == nil {
		return true
	}
	if rg.ignoreCase {
		s = strings.ToLower(s)
	}
	return rg.re.MatchString(s)
}

// Find returns a LineMatch for each line that matches rg in reader.
// LimitHit is true if some matches may not have been included in the result.
// NOTE: This is not safe to use concurrently.
func (rg *readerGrep) Find(zf *store.ZipFile, f *store.SrcFile) (matches []protocol.LineMatch, limitHit bool, err error) {
	// fileMatchBuf is what we run match on, fileBuf is the original
	// data (for Preview).
	fileBuf := zf.DataFor(f)
	fileMatchBuf := fileBuf

	// If we are ignoring case, we transform the input instead of
	// relying on the regular expression engine which can be
	// slow. compile has already lowercased the pattern. We also
	// trade some correctness for perf by using a non-utf8 aware
	// lowercase function.
	if rg.ignoreCase {
		if rg.transformBuf == nil {
			rg.transformBuf = make([]byte, zf.MaxLen)
		}
		fileMatchBuf = rg.transformBuf[:len(fileBuf)]
		bytesToLowerASCII(fileMatchBuf, fileBuf)
	}

	// Most files will not have a match and we bound the number of matched
	// files we return. So we can avoid the overhead of parsing out new lines
	// and repeatedly running the regex engine by running a single match over
	// the whole file. This does mean we duplicate work when actually
	// searching for results. We use the same approach when we search
	// per-line. Additionally if we have a non-empty literalSubstring, we use
	// that to prune out files since doing bytes.Index is very fast.
	if !bytes.Contains(fileMatchBuf, rg.literalSubstring) {
		return nil, false, nil
	}

	locs := rg.re.FindAllIndex(fileMatchBuf, maxLineMatches+1)
	lastStart := 0
	lastLineNumber := 0
	lastMatchIndex := 0
	lastLineStartIndex := 0

	for _, match := range locs {
		start, end := match[0], match[1]
		lineStart := lastLineStartIndex
		if idx := bytes.LastIndex(fileMatchBuf[lastStart:start], []byte{'\n'}); idx >= 0 {
			lineStart = lastStart + idx + 1
		}
		lastLineStartIndex = lineStart
		lastStart = start

		// lineEnd is the index of the next \n. If the last character of our
		// match is already a newline, then lineEnd instead points end to
		// include the newline in the match preview.
		var lineEnd int
		if end > 0 && fileMatchBuf[end-1] == '\n' {
			lineEnd = end // Note: fileMatchBuf[lineEnd] may not be a \n
		} else if idx := bytes.Index(fileMatchBuf[end:], []byte{'\n'}); idx >= 0 {
			lineEnd = end + idx
		} else {
			lineEnd = len(fileMatchBuf)
		}

		lineNumber, matchIndex := hydrateLineNumbers(fileMatchBuf, lastLineNumber, lastMatchIndex, lineStart, match)

		lastMatchIndex = matchIndex
		lastLineNumber = lineNumber
		matches = appendMatches(matches, fileBuf[lineStart:lineEnd], fileMatchBuf[lineStart:lineEnd], lineNumber, start-lineStart, end-lineStart)

		if len(matches) > maxLineMatches {
			matches = matches[:maxLineMatches]
			limitHit = true
			break
		}
	}
	return matches, limitHit, nil
}

func hydrateLineNumbers(fileBuf []byte, lastLineNumber, lastMatchIndex, lineStart int, match []int) (lineNumber, matchIndex int) {
	lineNumber = lastLineNumber + bytes.Count(fileBuf[lastMatchIndex:match[0]], []byte{'\n'})
	return lineNumber, lineStart
}

// matchLineBuf is a byte slice that contains the full line(s) that the match appears on.
func appendMatches(matches []protocol.LineMatch, fileBuf []byte, matchLineBuf []byte, lineNumber, start, end int) []protocol.LineMatch {
	// If any newlines appear between start and end, we need to append multiple LineMatch.
	// We assume there are no newlines before start.
	for len(matchLineBuf) > 0 {
		var line []byte
		var eol int
		if eol = bytes.Index(matchLineBuf[start:], []byte{'\n'}); eol < 0 {
			line = matchLineBuf
			matchLineBuf = []byte{}
		} else {
			eol += start
			// start is 0 indexed, so add 1 to include the new line at the end of the line
			line = matchLineBuf[:eol+1]
			matchLineBuf = matchLineBuf[eol+1:]
		}

		e := end
		if e > len(line) {
			e = len(line)
		}

		offset := utf8.RuneCount(line[:start])
		length := utf8.RuneCount(line[start:e])
		limit := eol
		if limit < 0 {
			limit = len(fileBuf)
		}
		matches = append(matches, protocol.LineMatch{
			// we are not allowed to use the fileBuf data after the ZipFile has been Closed,
			// which currently occurs before Preview has been serialized.
			// TODO: consider moving the call to Close until after we are
			// done with Preview, and stop making a copy here.
			// Special care must be taken to call Close on all possible paths, including error paths.
			Preview:          string(fileBuf[:limit]),
			LineNumber:       lineNumber,
			OffsetAndLengths: [][2]int{{offset, length}},
			LimitHit:         false, // We will always return false for this field since we no longer limit the number of offsets per line.
		})

		if eol >= 0 {
			fileBuf = fileBuf[eol+1:]
		}

		lineNumber++
		start = 0
		end -= e
	}
	return matches
}

// FindZip is a convenience function to run Find on f.
func (rg *readerGrep) FindZip(zf *store.ZipFile, f *store.SrcFile) (protocol.FileMatch, error) {
	lm, limitHit, err := rg.Find(zf, f)
	return protocol.FileMatch{
		Path:        f.Name,
		LineMatches: lm,
		MatchCount:  len(lm),
		LimitHit:    limitHit,
	}, err
}

// regexSearch concurrently searches files in zr looking for matches using rg.
func regexSearch(ctx context.Context, rg *readerGrep, zf *store.ZipFile, fileMatchLimit int, patternMatchesContent, patternMatchesPaths bool, isPatternNegated bool) (fm []protocol.FileMatch, limitHit bool, err error) {
	span, ctx := ot.StartSpanFromContext(ctx, "RegexSearch")
	ext.Component.Set(span, "regex_search")
	if rg.re != nil {
		span.SetTag("re", rg.re.String())
	}
	span.SetTag("path", rg.matchPath.String())
	defer func() {
		if err != nil {
			ext.Error.Set(span, true)
			span.SetTag("err", err.Error())
		}
		span.Finish()
	}()

	if !patternMatchesContent && !patternMatchesPaths {
		patternMatchesContent = true
	}

	if fileMatchLimit > maxFileMatches || fileMatchLimit <= 0 {
		fileMatchLimit = maxFileMatches
	}

	// If we reach fileMatchLimit we use cancel to stop the search
	var cancel context.CancelFunc
	if deadline, ok := ctx.Deadline(); ok {
		// If a deadline is set, try to finish before the deadline expires.
		timeout := time.Duration(0.9 * float64(time.Until(deadline)))
		span.LogFields(otlog.Int64("RegexSearchTimeout", int64(timeout)))
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	var (
		filesmu   sync.Mutex // protects files
		files     = zf.Files
		matchesmu sync.Mutex // protects matches, limitHit
		matches   = []protocol.FileMatch{}
	)

	if rg.re == nil || (patternMatchesPaths && !patternMatchesContent) {
		// Fast path for only matching file paths (or with a nil pattern, which matches all files,
		// so is effectively matching only on file paths).
		for _, f := range files {
			if match := rg.matchPath.MatchPath(f.Name) && rg.matchString(f.Name); match == !isPatternNegated {
				if len(matches) < fileMatchLimit {
					matches = append(matches, protocol.FileMatch{Path: f.Name})
				} else {
					limitHit = true
					break
				}
			}
		}
		return matches, limitHit, nil
	}

	var (
		done          = ctx.Done()
		wg            sync.WaitGroup
		wgErrOnce     sync.Once
		wgErr         error
		filesSkipped  uint32 // accessed atomically
		filesSearched uint32 // accessed atomically
	)

	// Start workers. They read from files and write to matches.
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(rg *readerGrep) {
			defer wg.Done()

			for {
				// check whether we've been cancelled
				select {
				case <-done:
					return
				default:
				}

				// grab a file to work on
				filesmu.Lock()
				if len(files) == 0 {
					filesmu.Unlock()
					return
				}
				f := &files[0]
				files = files[1:]
				filesmu.Unlock()

				// decide whether to process, record that decision
				if !rg.matchPath.MatchPath(f.Name) {
					atomic.AddUint32(&filesSkipped, 1)
					continue
				}
				atomic.AddUint32(&filesSearched, 1)

				// process
				var fm protocol.FileMatch
				fm, err := rg.FindZip(zf, f)
				if err != nil {
					wgErrOnce.Do(func() {
						wgErr = err
						cancel()
					})
					return
				}
				match := len(fm.LineMatches) > 0
				if !match && patternMatchesPaths {
					// Try matching against the file path.
					match = rg.matchString(f.Name)
					if match {
						fm.Path = f.Name
					}
				}
				if match == !isPatternNegated {
					matchesmu.Lock()
					if len(matches) < fileMatchLimit {
						matches = append(matches, fm)
					} else {
						limitHit = true
						cancel()
					}
					matchesmu.Unlock()
				}
			}
		}(rg.Copy())
	}

	wg.Wait()

	err = wgErr
	if err == nil && ctx.Err() == context.DeadlineExceeded {
		// We stopped early because we were about to hit the deadline.
		err = ctx.Err()
	}

	span.LogFields(
		otlog.Int("filesSkipped", int(atomic.LoadUint32(&filesSkipped))),
		otlog.Int("filesSearched", int(atomic.LoadUint32(&filesSearched))),
	)

	return matches, limitHit, err
}

// lowerRegexpASCII lowers rune literals and expands char classes to include
// lowercase. It does it inplace. We can't just use strings.ToLower since it
// will change the meaning of regex shorthands like \S or \B.
func lowerRegexpASCII(re *syntax.Regexp) {
	for _, c := range re.Sub {
		if c != nil {
			lowerRegexpASCII(c)
		}
	}
	switch re.Op {
	case syntax.OpLiteral:
		// For literal strings we can simplify lower each character.
		for i := range re.Rune {
			re.Rune[i] = unicode.ToLower(re.Rune[i])
		}
	case syntax.OpCharClass:
		l := len(re.Rune)

		// An exclusion class is something like [^A-Z]. We need to specially
		// handle it since the user intention of [^A-Z] should map to
		// [^a-z]. If we use the normal mapping logic, we will do nothing
		// since [a-z] is in [^A-Z]. We assume we have an exclusion class if
		// our inclusive range starts at 0 and ends at the end of the unicode
		// range. Note this means we don't support unusual ranges like
		// [^\x00-B] or [^B-\x{10ffff}].
		isExclusion := l >= 4 && re.Rune[0] == 0 && re.Rune[l-1] == utf8.MaxRune
		if isExclusion {
			// Algorithm:
			// Assume re.Rune is sorted (it is!)
			// 1. Build a list of inclusive ranges in a-z that are excluded in A-Z (excluded)
			// 2. Copy across classes, ensuring all ranges are outside of ranges in excluded.
			//
			// In our comments we use the mathematical notation [x, y] and (a,
			// b). [ and ] are range inclusive, ( and ) are range
			// exclusive. So x is in [x, y], but not in (x, y).

			// excluded is a list of _exclusive_ ranges in ['a', 'z'] that need
			// to be removed.
			excluded := []rune{}

			// Note i starts at 1, so we are inspecting the gaps between
			// ranges. So [re.Rune[0], re.Rune[1]] and [re.Rune[2],
			// re.Rune[3]] impiles we have an excluded range of (re.Rune[1],
			// re.Rune[2]).
			for i := 1; i < l-1; i += 2 {
				// (a, b) is a range that is excluded
				a, b := re.Rune[i], re.Rune[i+1]
				// This range doesn't exclude [A-Z], so skip (does not
				// intersect with ['A', 'Z']).
				if a > 'Z' || b < 'A' {
					continue
				}
				// We know (a, b) intersects with ['A', 'Z']. So clamp such
				// that we have the intersection (a, b) ^ [A, Z]
				if a < 'A' {
					a = 'A' - 1
				}
				if b > 'Z' {
					b = 'Z' + 1
				}
				// (a, b) is now a range contained in ['A', 'Z'] that needs to
				// be excluded. So we map it to the lower case version and add
				// it to the excluded list.
				excluded = append(excluded, a+'a'-'A', b+'b'-'B')
			}

			// Adjust re.Rune to exclude excluded. This may require shrinking
			// or growing the list, so we do it to a copy.
			copy := make([]rune, 0, len(re.Rune))
			for i := 0; i < l; i += 2 {
				// [a, b] is a range that is included
				a, b := re.Rune[i], re.Rune[i+1]

				// Remove exclusions ranges that occur before a. They would of
				// been previously processed.
				for len(excluded) > 0 && a >= excluded[1] {
					excluded = excluded[2:]
				}

				// If our exclusion range happens after b, that means we
				// should only consider it later.
				if len(excluded) == 0 || b <= excluded[0] {
					copy = append(copy, a, b)
					continue
				}

				// We now know that the current exclusion range intersects
				// with [a, b]. Break it into two parts, the range before a
				// and the range after b.
				if a <= excluded[0] {
					copy = append(copy, a, excluded[0])
				}
				if b >= excluded[1] {
					copy = append(copy, excluded[1], b)
				}
			}
			re.Rune = copy
		} else {
			for i := 0; i < l; i += 2 {
				// We found a char class that includes a-z. No need to
				// modify.
				if re.Rune[i] <= 'a' && re.Rune[i+1] >= 'z' {
					return
				}
			}
			for i := 0; i < l; i += 2 {
				a, b := re.Rune[i], re.Rune[i+1]
				// This range doesn't include A-Z, so skip
				if a > 'Z' || b < 'A' {
					continue
				}
				simple := true
				if a < 'A' {
					simple = false
					a = 'A'
				}
				if b > 'Z' {
					simple = false
					b = 'Z'
				}
				a, b = unicode.ToLower(a), unicode.ToLower(b)
				if simple {
					// The char range is within A-Z, so we can
					// just modify it to be the equivalent in a-z.
					re.Rune[i], re.Rune[i+1] = a, b
				} else {
					// The char range includes characters outside
					// of A-Z. To be safe we just append a new
					// lowered range which is the intersection
					// with A-Z.
					re.Rune = append(re.Rune, a, b)
				}
			}
		}
	default:
		return
	}
	// Copy to small storage if necessary
	for i := 0; i < 2 && i < len(re.Rune); i++ {
		re.Rune0[i] = re.Rune[i]
	}
}

// longestLiteral finds the longest substring that is guaranteed to appear in
// a match of re.
//
// Note: There may be a longer substring that is guaranteed to appear. For
// example we do not find the longest common substring in alternating
// group. Nor do we handle concatting simple capturing groups.
func longestLiteral(re *syntax.Regexp) string {
	switch re.Op {
	case syntax.OpLiteral:
		return string(re.Rune)
	case syntax.OpCapture, syntax.OpPlus:
		return longestLiteral(re.Sub[0])
	case syntax.OpRepeat:
		if re.Min >= 1 {
			return longestLiteral(re.Sub[0])
		}
	case syntax.OpConcat:
		longest := ""
		for _, sub := range re.Sub {
			l := longestLiteral(sub)
			if len(l) > len(longest) {
				longest = l
			}
		}
		return longest
	}
	return ""
}

// readAll will read r until EOF into b. It returns the number of bytes
// read. If we do not reach EOF, an error is returned.
func readAll(r io.Reader, b []byte) (int, error) {
	n := 0
	for {
		if len(b) == 0 {
			// We may be at EOF, but it hasn't returned that
			// yet. Technically r.Read is allowed to return 0,
			// nil, but it is strongly discouraged. If they do, we
			// will just return an err.
			scratch := []byte{'1'}
			_, err := r.Read(scratch)
			if err == io.EOF {
				return n, nil
			}
			return n, errors.New("reader is too large")
		}

		m, err := r.Read(b)
		n += m
		b = b[m:]
		if err != nil {
			if err == io.EOF { // done
				return n, nil
			}
			return n, err
		}
	}
}
