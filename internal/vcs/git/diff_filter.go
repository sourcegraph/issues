package git

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"regexp"
	"runtime/debug"
	"unicode/utf8"

	"github.com/sourcegraph/go-diff/diff"
	"github.com/sourcegraph/sourcegraph/internal/pathmatch"
)

// compilePathMatcher compiles the path options into a PathMatcher.
func compilePathMatcher(options PathOptions) (pathmatch.PathMatcher, error) {
	return pathmatch.CompilePathPatterns(
		options.IncludePatterns, options.ExcludePattern,
		pathmatch.CompileOptions{CaseSensitive: options.IsCaseSensitive, RegExp: options.IsRegExp},
	)
}

// filterAndHighlightDiff returns the raw diff with query matches highlighted
// and only hunks that satisfy the query (if onlyMatchingHunks) and path matcher.
func filterAndHighlightDiff(rawDiff []byte, query *regexp.Regexp, onlyMatchingHunks bool, pathMatcher pathmatch.PathMatcher) (_ []byte, _ []Highlight, err error) {
	// go-diff has been known to panic. Until we are sure it has been written
	// to avoid panics, we protect calles from the panic. eg
	// https://github.com/sourcegraph/go-diff/issues/54
	defer func() {
		if panicValue := recover(); panicValue != nil {
			stack := debug.Stack()
			log.Printf("filterAndHighlightDiff panic: %v\n%s", panicValue, stack)
			err = fmt.Errorf("filterAndHighlightDiff panic: %v", panicValue)
		}
	}()

	const (
		maxFiles          = 5
		maxHunksPerFile   = 3
		matchContextLines = 1
		maxLinesPerHunk   = 5
		maxMatchesPerLine = 100
		maxCharsPerLine   = 200
	)

	dr := diff.NewMultiFileDiffReader(bytes.NewReader(rawDiff))
	var matchingFileDiffs []*diff.FileDiff
	for i := 0; ; i++ {
		fileDiff, err := dr.ReadFile()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, nil, err
		}

		// Exclude files whose names don't match.
		origNameMatches := fileDiff.OrigName != "/dev/null" && pathMatcher.MatchPath(fileDiff.OrigName)
		newNameMatches := fileDiff.NewName != "/dev/null" && pathMatcher.MatchPath(fileDiff.NewName)
		if !origNameMatches && !newNameMatches {
			continue
		}

		// TODO(sqs): preserve the "no newline" message. We clear it out because our truncateLongLines
		// and splitHunkMatches funcs don't properly adjust its offset as they modify hunk.Body. If
		// the OrigNoNewlineAt points to an out-of-bounds offset, a panic will occur.
		for _, hunk := range fileDiff.Hunks {
			hunk.OrigNoNewlineAt = 0
		}

		// Truncate long lines, for perf.
		for _, hunk := range fileDiff.Hunks {
			hunk.Body = truncateLongLines(hunk.Body, maxCharsPerLine)
		}

		// Exclude hunks not matching the query.
		if onlyMatchingHunks {
			fileDiff.Hunks = splitHunkMatches(fileDiff.Hunks, query, matchContextLines, maxLinesPerHunk)
		}

		if len(fileDiff.Hunks) > 0 {
			if len(fileDiff.Hunks) > maxHunksPerFile {
				fileDiff.Hunks = fileDiff.Hunks[:maxHunksPerFile]
			}
			matchingFileDiffs = append(matchingFileDiffs, fileDiff)
		}
	}

	if len(matchingFileDiffs) > maxFiles {
		matchingFileDiffs = matchingFileDiffs[:maxFiles]
	} else if len(matchingFileDiffs) == 0 {
		return nil, nil, nil
	}

	rawDiff, err = diff.PrintMultiFileDiff(matchingFileDiffs)
	if err != nil {
		return nil, nil, err
	}

	// Highlight query matches in raw diff.
	var highlights []Highlight
	ignoreUntilAfterAtAt := false
	for i, line := range bytes.Split(rawDiff, []byte("\n")) {
		// Don't highlight matches that are not in the body (such as "+++ file1" or "--- file1").
		if ignoreUntilAfterAtAt {
			if atAt := bytes.HasPrefix(line, []byte("@@")); atAt {
				ignoreUntilAfterAtAt = false
				continue
			}
		}
		if bytes.HasPrefix(line, []byte("diff ")) {
			ignoreUntilAfterAtAt = true
			continue
		}

		// Always ignore subsequent "@@" lines, which are hunk headers.
		if bytes.HasPrefix(line, []byte("@@")) {
			continue
		}

		if len(line) == 0 {
			continue
		}
		if query != nil {
			lineWithoutStatus := line[1:] // don't match '-' or '+' line status
			for _, match := range query.FindAllIndex(lineWithoutStatus, maxMatchesPerLine) {
				highlights = append(highlights, Highlight{
					Line:      i + 1,
					Character: match[0] + 1,
					Length:    match[1] - match[0],
				})
			}
		}
	}

	return rawDiff, highlights, nil
}

func truncateLongLines(data []byte, maxCharsPerLine int) []byte {
	// We reuse data's storage to avoid allocation.

	offset := 0
	lineLength := 0
	hasSeenTruncatedLine := false // avoid writing until we need to
	r := bytes.NewReader(data)
	for {
		r, n, err := r.ReadRune()
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
			break
		}
		if r == '\n' {
			lineLength = -1 // will be incremented immediately after to 0
		}
		if lineLength < maxCharsPerLine {
			if hasSeenTruncatedLine {
				utf8.EncodeRune(data[offset:], r)
			}
			offset += n
			lineLength++
		} else {
			hasSeenTruncatedLine = true
		}
	}

	return data[:offset]
}

func diffHunkLineStatus(line []byte) (added, removed bool) {
	if len(line) >= 1 {
		added = line[0] == '+'
		removed = line[0] == '-'
	}
	return
}

type diffHunkLineInfo struct {
	added    bool // line starts with '+'
	removed  bool // line starts with '-'
	matching bool // line matches query (only computed for changed lines)
	context  bool // include because it's context for a matching changed line
}

func (info diffHunkLineInfo) changed() bool { return info.added || info.removed }

func computeDiffHunkInfo(lines [][]byte, query *regexp.Regexp, matchContextLines int) []diffHunkLineInfo {
	// Return context line numbers for a given line number.
	contextLines := func(line int) (start, end int) {
		start = line - matchContextLines
		if start < 0 {
			start = 0
		}
		end = line + matchContextLines
		if end > len(lines)-1 {
			end = len(lines) - 1
		}
		return
	}

	lineInfo := make([]diffHunkLineInfo, len(lines))
	for i, line := range lines {
		lineInfo[i].added, lineInfo[i].removed = diffHunkLineStatus(line)
		if lineInfo[i].changed() {
			lineInfo[i].matching = query == nil || query.Match(line)
			if lineInfo[i].matching {
				// Mark context lines before/after matching lines.
				start, end := contextLines(i)
				for j := start; j <= end; j++ {
					if i == j {
						continue
					}
					lineInfo[j].context = true
				}
			}
		}
	}
	return lineInfo
}

// splitHunkMatches returns a list of hunks that are a subset of the input hunks,
// filtered down to only hunks that match the query. Non-matching context lines
// and non-matching changed lines are eliminated, and the hunk header (start/end
// lines) are adjusted accordingly.
func splitHunkMatches(hunks []*diff.Hunk, query *regexp.Regexp, matchContextLines, maxLinesPerHunk int) (results []*diff.Hunk) {
	addExtraHunkMatchesSection := func(hunk *diff.Hunk, extraHunkMatches int) {
		if extraHunkMatches > 0 {
			if hunk.Section != "" {
				hunk.Section += " "
			}
			hunk.Section += fmt.Sprintf("... +%d", extraHunkMatches)
		}
	}

	for _, hunk := range hunks {
		var cur *diff.Hunk
		var curLines [][]byte

		lines := bytes.SplitAfter(hunk.Body, []byte("\n"))
		lineInfo := computeDiffHunkInfo(lines, query, matchContextLines)

		extraHunkMatches := 0
		var origLineOffset, newLineOffset int32
		for i, line := range lines {
			if len(line) == 0 {
				break
			}
			lineInfo := lineInfo[i]
			if lineInfo.matching || lineInfo.context {
				if maxLinesPerHunk > 0 && len(curLines) == matchContextLines*2+maxLinesPerHunk {
					if lineInfo.matching {
						extraHunkMatches++
					}
					continue
				}

				if cur == nil {
					cur = &diff.Hunk{
						OrigStartLine: hunk.OrigStartLine + origLineOffset,
						NewStartLine:  hunk.NewStartLine + newLineOffset,
						Section:       hunk.Section,
					}
				}

				if !lineInfo.added {
					cur.OrigLines++
				}
				if !lineInfo.removed {
					cur.NewLines++
				}
				curLines = append(curLines, line)
			} else if cur != nil {
				addExtraHunkMatchesSection(cur, extraHunkMatches)
				cur.Body = bytes.Join(curLines, nil)
				results = append(results, cur)
				cur = nil
				curLines = nil
				extraHunkMatches = 0
			}

			if !lineInfo.added {
				origLineOffset++
			}
			if !lineInfo.removed {
				newLineOffset++
			}
		}

		if cur != nil {
			addExtraHunkMatchesSection(cur, extraHunkMatches)
			cur.Body = bytes.Join(curLines, nil)
			results = append(results, cur)
		}
	}

	return results
}
