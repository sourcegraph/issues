package resolvers

import (
	"bytes"
	"context"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/sourcegraph/go-diff/diff"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

// PositionAdjuster translates a position within a git tree at a source commit into the
// equivalent position in a target commit commit. The position adjuster instance carries
// along with it the source commit.
type PositionAdjuster interface {
	// AdjustPath translates the given path from the source commit into the given target
	// commit. If revese is true, then the source and target commits are swapped.
	AdjustPath(ctx context.Context, commit, path string, reverse bool) (string, bool, error)

	// AdjustPosition translates the given position from the source commit into the given
	// target commit. The adjusted path and position are returned, along with a boolean flag
	// indicating that the translation was successful. If revese is true, then the source and
	// target commits are swapped.
	AdjustPosition(ctx context.Context, commit, path string, px bundles.Position, reverse bool) (string, bundles.Position, bool, error)

	// AdjustRange translates the given range from the source commit into the given target
	// commit. The adjusted path and range are returned, along with a boolean flag indicating
	// that the translation was successful. If revese is true, then the source and target commits
	// are swapped.
	AdjustRange(ctx context.Context, commit, path string, rx bundles.Range, reverse bool) (string, bundles.Range, bool, error)
}

type positionAdjuster struct {
	repo      *types.Repo
	commit    string
	hunkCache HunkCache
}

// NewPositionAdjuster creates a new PositionAdjuster with the given repository and source commit.
func NewPositionAdjuster(repo *types.Repo, commit string, hunkCache HunkCache) PositionAdjuster {
	return &positionAdjuster{
		repo:      repo,
		commit:    commit,
		hunkCache: hunkCache,
	}
}

// AdjustPath translates the given path from the source commit into the given target
// commit. If revese is true, then the source and target commits are swapped.
func (p *positionAdjuster) AdjustPath(ctx context.Context, commit, path string, reverse bool) (string, bool, error) {
	return path, true, nil
}

// AdjustPosition translates the given position from the source commit into the given
// target commit. The adjusted path and position are returned, along with a boolean flag
// indicating that the translation was successful. If revese is true, then the source and
// target commits are swapped.
func (p *positionAdjuster) AdjustPosition(ctx context.Context, commit, path string, px bundles.Position, reverse bool) (string, bundles.Position, bool, error) {
	hunks, err := p.readHunksCached(ctx, p.repo, p.commit, commit, path, reverse)
	if err != nil {
		return "", bundles.Position{}, false, err
	}

	adjusted, ok := adjustPosition(hunks, px)
	return path, adjusted, ok, nil
}

// AdjustRange translates the given range from the source commit into the given target
// commit. The adjusted path and range are returned, along with a boolean flag indicating
// that the translation was successful. If revese is true, then the source and target commits
// are swapped.
func (p *positionAdjuster) AdjustRange(ctx context.Context, commit, path string, rx bundles.Range, reverse bool) (string, bundles.Range, bool, error) {
	hunks, err := p.readHunksCached(ctx, p.repo, p.commit, commit, path, reverse)
	if err != nil {
		return "", bundles.Range{}, false, err
	}

	adjusted, ok := adjustRange(hunks, rx)
	return path, adjusted, ok, nil
}

// readHunksCached returns a position-ordered slice of changes (additions or deletions) of
// the given path between the given source and target commits. If revese is true, then the
// source and target commits are swapped. If the position adjuster has a hunk cache, it
// will read from it before attempting to contact a remote server, and populate the cache
// with new results
func (p *positionAdjuster) readHunksCached(ctx context.Context, repo *types.Repo, sourceCommit, targetCommit, path string, reverse bool) ([]*diff.Hunk, error) {
	if sourceCommit == targetCommit {
		return nil, nil
	}
	if reverse {
		sourceCommit, targetCommit = targetCommit, sourceCommit
	}

	if p.hunkCache == nil {
		return p.readHunks(ctx, repo, sourceCommit, targetCommit, path)
	}

	key := makeKey(strconv.FormatInt(int64(repo.ID), 10), sourceCommit, targetCommit, path)
	if hunks, ok := p.hunkCache.Get(key); ok {
		if hunks == nil {
			return nil, nil
		}

		return hunks.([]*diff.Hunk), nil
	}

	hunks, err := p.readHunks(ctx, repo, sourceCommit, targetCommit, path)
	if err != nil {
		return nil, err
	}

	p.hunkCache.Set(key, hunks, int64(len(hunks)))

	return hunks, nil
}

// readHunks returns a position-ordered slice of changes (additions or deletions) of
// the given path between the given source and target commits.
func (p *positionAdjuster) readHunks(ctx context.Context, repo *types.Repo, sourceCommit, targetCommit, path string) ([]*diff.Hunk, error) {
	cachedRepo, err := backend.CachedGitRepo(ctx, repo)
	if err != nil {
		return nil, err
	}

	reader, err := git.ExecReader(ctx, *cachedRepo, []string{"diff", sourceCommit, targetCommit, "--", path})
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	output, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if len(output) == 0 {
		return nil, nil
	}

	diff, err := diff.NewFileDiffReader(bytes.NewReader(output)).Read()
	if err != nil {
		return nil, err
	}
	return diff.Hunks, nil
}

// adjustPosition translates the given position by adjusting the line number based on the
// number of additions and deletions that occur before that line. This function returns a
// boolean flag indicating that the translation is successful. A translation fails when the
// line indicated by the position has been edited.
func adjustPosition(hunks []*diff.Hunk, pos bundles.Position) (bundles.Position, bool) {
	line, ok := adjustLine(hunks, pos.Line)
	if !ok {
		return bundles.Position{}, false
	}

	return bundles.Position{Line: line, Character: pos.Character}, true
}

// adjustLine translates the given line nubmerbased on the number of additions and deletions
// that occur before that line. This function returns a boolean flag indicating that the
// translation is successful. A translation fails when the given line has been edited.
func adjustLine(hunks []*diff.Hunk, line int) (int, bool) {
	// Translate from bundle/lsp zero-index to git diff one-index
	line = line + 1

	hunk := findHunk(hunks, line)
	if hunk == nil {
		// Trivial case, no changes before this line
		return line - 1, true
	}

	// If the hunk ends before this line, we can simply adjust the line offset by the
	// relative difference between the line offsets in each file after this hunk.
	if line >= int(hunk.OrigStartLine+hunk.OrigLines) {
		endOfSourceHunk := int(hunk.OrigStartLine + hunk.OrigLines)
		endOfTargetHunk := int(hunk.NewStartLine + hunk.NewLines)
		adjustedLine := line + (endOfTargetHunk - endOfSourceHunk)

		// Translate from git diff one-index to bundle/lsp zero-index
		return adjustedLine - 1, true
	}

	// These offsets start at the beginning of the hunk's delta. The following loop will
	// process the delta line-by-line. For each line that exists the source (orig) or
	// target (new) file, the corresponding offset will be bumped. The values of these
	// offsets once we hit our target line will determine the relative offset between
	// the two files.
	sourceOffset := int(hunk.OrigStartLine)
	targetOffset := int(hunk.NewStartLine)

	for _, deltaLine := range strings.Split(string(hunk.Body), "\n") {
		isAdded := strings.HasPrefix(deltaLine, "+")
		isRemoved := strings.HasPrefix(deltaLine, "-")

		// A line exists in the source file if it wasn't added in the delta. We adjust
		// this before the next condition so that our comparison with our target line
		// is correct.
		if !isAdded {
			sourceOffset++
		}

		// Hit our target line
		if sourceOffset-1 == line {
			// This particular line was (1) edited; (2) removed, or (3) added.
			// If it was removed, there is nothing to point to in the target file.
			// If it was added, then we don't have any index information for it in
			// our source file. In any case, we won't have a precise translation.
			if isAdded || isRemoved {
				return 0, false
			}

			// Translate from git diff one-index to bundle/lsp zero-index
			return targetOffset - 1, true
		}

		// A line exists in the target file if it wasn't deleted in the delta. We adjust
		// this after the previous condition so we don't have to re-adjust the target offset
		// within the exit conditions (this adjustment is only necessary for future iterations).
		if !isRemoved {
			targetOffset++
		}
	}

	// This should never happen unless the git diff content is malformed. We know
	// the target line occurs within the hunk, but iteration of the hunk's body did
	// not contain enough lines attributed to the original file.
	panic("Malformed hunk body")
}

// findHunk returns the last thunk that does not begin after the given line.
func findHunk(hunks []*diff.Hunk, line int) *diff.Hunk {
	i := 0
	for i < len(hunks) && int(hunks[i].OrigStartLine) <= line {
		i++
	}

	if i == 0 {
		return nil
	}
	return hunks[i-1]
}

// adjustRange translates the given range by calling adjustPosition on both of hte range's
// endpoints. This function returns a boolean flag indicating that the translation was
// successful (which occurs when both endpoints of the range can be translated).
func adjustRange(hunks []*diff.Hunk, r bundles.Range) (bundles.Range, bool) {
	start, ok := adjustPosition(hunks, r.Start)
	if !ok {
		return bundles.Range{}, false
	}

	end, ok := adjustPosition(hunks, r.End)
	if !ok {
		return bundles.Range{}, false
	}

	return bundles.Range{Start: start, End: end}, true
}

func makeKey(parts ...string) string {
	return strings.Join(parts, ":")
}
