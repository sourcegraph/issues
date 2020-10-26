package graphqlbackend

import (
	"context"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/sourcegraph/go-diff/diff"
)

type PreviewRepositoryComparisonResolver interface {
	RepositoryComparisonInterface
}

// NewPreviewRepositoryComparisonResolver is a convenience function to get a preview diff from a repo, given a base rev and the git patch.
func NewPreviewRepositoryComparisonResolver(ctx context.Context, repo *RepositoryResolver, baseRev, patch string) (*previewRepositoryComparisonResolver, error) {
	args := &RepositoryCommitArgs{Rev: baseRev}
	commit, err := repo.Commit(ctx, args)
	if err != nil {
		return nil, err
	}
	return &previewRepositoryComparisonResolver{
		repo:   repo,
		commit: commit,
		patch:  patch,
	}, nil
}

type previewRepositoryComparisonResolver struct {
	repo   *RepositoryResolver
	commit *GitCommitResolver
	patch  string
}

// Type guard.
var _ RepositoryComparisonInterface = &previewRepositoryComparisonResolver{}

func (r *previewRepositoryComparisonResolver) ToPreviewRepositoryComparison() (PreviewRepositoryComparisonResolver, bool) {
	return r, true
}

func (r *previewRepositoryComparisonResolver) ToRepositoryComparison() (*RepositoryComparisonResolver, bool) {
	return nil, false
}

func (r *previewRepositoryComparisonResolver) BaseRepository() *RepositoryResolver {
	return r.repo
}

func (r *previewRepositoryComparisonResolver) FileDiffs(ctx context.Context, args *FileDiffsConnectionArgs) (FileDiffConnection, error) {
	return NewFileDiffConnectionResolver(r.commit, r.commit, args, fileDiffConnectionCompute(r.patch), previewNewFile), nil
}

func fileDiffConnectionCompute(patch string) func(ctx context.Context, args *FileDiffsConnectionArgs) ([]*diff.FileDiff, int32, bool, error) {
	var (
		once        sync.Once
		fileDiffs   []*diff.FileDiff
		afterIdx    int32
		hasNextPage bool
		err         error
	)
	return func(ctx context.Context, args *FileDiffsConnectionArgs) ([]*diff.FileDiff, int32, bool, error) {
		once.Do(func() {
			if args.After != nil {
				parsedIdx, err := strconv.ParseInt(*args.After, 0, 32)
				if err != nil {
					return
				}
				if parsedIdx < 0 {
					parsedIdx = 0
				}
				afterIdx = int32(parsedIdx)
			}
			totalAmount := afterIdx
			if args.First != nil {
				totalAmount += *args.First
			}

			dr := diff.NewMultiFileDiffReader(strings.NewReader(patch))
			for {
				var fileDiff *diff.FileDiff
				fileDiff, err = dr.ReadFile()
				if err == io.EOF {
					err = nil
					break
				}
				if err != nil {
					return
				}
				fileDiffs = append(fileDiffs, fileDiff)
				if len(fileDiffs) == int(totalAmount) {
					// Check for hasNextPage.
					_, err = dr.ReadFile()
					if err != nil && err != io.EOF {
						return
					}
					if err == io.EOF {
						err = nil
					} else {
						hasNextPage = true
					}
					break
				}
			}
		})
		return fileDiffs, afterIdx, hasNextPage, err
	}
}

func previewNewFile(r *FileDiffResolver) FileResolver {
	fileStat := CreateFileInfo(r.FileDiff.NewName, false)
	return NewVirtualFileResolver(fileStat, fileDiffVirtualFileContent(r))
}

func fileDiffVirtualFileContent(r *FileDiffResolver) FileContentFunc {
	var (
		once       sync.Once
		newContent string
		err        error
	)
	return func(ctx context.Context) (string, error) {
		once.Do(func() {
			var oldContent string
			if oldFile := r.OldFile(); oldFile != nil {
				var err error
				oldContent, err = r.OldFile().Content(ctx)
				if err != nil {
					return
				}
			}
			newContent = applyPatch(oldContent, r.FileDiff)
		})
		return newContent, err
	}
}

func applyPatch(fileContent string, fileDiff *diff.FileDiff) string {
	if diffPathOrNull(fileDiff.NewName) == nil {
		// the file was deleted, no need to do costly computation.
		return ""
	}
	addNewline := true
	contentLines := strings.Split(fileContent, "\n")
	newContentLines := make([]string, 0)
	var lastLine int32 = 1
	// Assumes the hunks are sorted by ascending lines.
	for _, hunk := range fileDiff.Hunks {
		if hunk.OrigNoNewlineAt > 0 {
			addNewline = true
		}
		// Detect holes.
		if hunk.OrigStartLine != 0 && hunk.OrigStartLine != lastLine {
			originalLines := contentLines[lastLine-1 : hunk.OrigStartLine-1]
			newContentLines = append(newContentLines, originalLines...)
			lastLine += int32(len(originalLines))
		}
		hunkLines := strings.Split(string(hunk.Body), "\n")
		for _, line := range hunkLines {
			switch {
			case line == "":
				// Ignore empty lines, they just indicate the last line of a hunk.
			case strings.HasPrefix(line, "-"):
				lastLine++
			case strings.HasPrefix(line, "+"):
				newContentLines = append(newContentLines, line[1:])
			default:
				newContentLines = append(newContentLines, contentLines[lastLine-1])
				lastLine++
			}
		}
	}
	// Append remaining lines from original file.
	if origLines := int32(len(contentLines)); origLines > 0 && origLines != lastLine {
		newContentLines = append(newContentLines, contentLines[lastLine-1:]...)
	} else {
		content := strings.Join(newContentLines, "\n")
		if addNewline {
			// If the file has a final newline character, we need to append it again.
			content += "\n"
		}
		return content
	}
	return strings.Join(newContentLines, "\n")
}
