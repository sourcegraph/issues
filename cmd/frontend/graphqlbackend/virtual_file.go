package graphqlbackend

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/externallink"
	"github.com/sourcegraph/sourcegraph/internal/highlight"
)

// FileContentFunc is a closure that returns the contents of a file and is used by the VirtualFileResolver.
type FileContentFunc func(ctx context.Context) (string, error)

func NewVirtualFileResolver(stat os.FileInfo, fileContent FileContentFunc) *virtualFileResolver {
	return &virtualFileResolver{
		stat:        stat,
		fileContent: fileContent,
	}
}

type virtualFileResolver struct {
	fileContent FileContentFunc
	// stat is this tree entry's file info. Its Name method must return the full path relative to
	// the root, not the basename.
	stat os.FileInfo
}

func (r *virtualFileResolver) Path() string      { return r.stat.Name() }
func (r *virtualFileResolver) Name() string      { return path.Base(r.stat.Name()) }
func (r *virtualFileResolver) IsDirectory() bool { return r.stat.Mode().IsDir() }

func (r *virtualFileResolver) ToGitBlob() (*GitTreeEntryResolver, bool)    { return nil, false }
func (r *virtualFileResolver) ToVirtualFile() (*virtualFileResolver, bool) { return r, true }

func (r *virtualFileResolver) URL(ctx context.Context) (string, error) {
	// Todo: allow viewing arbitrary files in the webapp.
	return "", nil
}

func (r *virtualFileResolver) CanonicalURL() (string, error) {
	// Todo: allow viewing arbitrary files in the webapp.
	return "", nil
}

func (r *virtualFileResolver) ExternalURLs(ctx context.Context) ([]*externallink.Resolver, error) {
	// Todo: allow viewing arbitrary files in the webapp.
	return []*externallink.Resolver{}, nil
}

func (r *virtualFileResolver) Content(ctx context.Context) (string, error) {
	return r.fileContent(ctx)
}

func (r *virtualFileResolver) RichHTML(ctx context.Context) (string, error) {
	content, err := r.Content(ctx)
	if err != nil {
		return "", err
	}
	return richHTML(content, path.Ext(r.Path()))
}

func (r *virtualFileResolver) Binary(ctx context.Context) (bool, error) {
	content, err := r.Content(ctx)
	if err != nil {
		return false, err
	}
	return highlight.IsBinary([]byte(content)), nil
}

func (r *virtualFileResolver) Highlight(ctx context.Context, args *HighlightArgs) (*highlightedFileResolver, error) {
	content, err := r.Content(ctx)
	if err != nil {
		return nil, err
	}
	return highlightContent(ctx, args, content, r.Path(), highlight.Metadata{
		Revision: fmt.Sprintf("Preview file diff %s", r.stat.Name()),
	})
}
