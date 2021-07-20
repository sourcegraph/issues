package app

import (
	"context"
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/token"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/sourcegraph/ctxvfs"
	"github.com/sourcegraph/go-lsp"
	"golang.org/x/tools/go/buildutil"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/gosrc"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/vfsutil"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
)

// serveGoSymbolURL handles Go symbol URLs (e.g.,
// https://sourcegraph.com/go/github.com/gorilla/mux/-/Vars) by
// redirecting them to the file and line/column URL of the definition.
func serveGoSymbolURL(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	spec, err := parseGoSymbolURLPath(r.URL.Path)
	if err != nil {
		return err
	}

	dir, err := gosrc.ResolveImportPath(httpcli.ExternalDoer(), spec.Pkg)
	if err != nil {
		return err
	}
	cloneURL := dir.CloneURL

	if cloneURL == "" || !strings.HasPrefix(cloneURL, "https://github.com") {
		return errors.Errorf("non-github clone URL resolved for import path %s", spec.Pkg)
	}

	repoName := api.RepoName(strings.TrimSuffix(strings.TrimPrefix(cloneURL, "https://"), ".git"))
	repo, err := backend.Repos.GetByName(ctx, repoName)
	if err != nil {
		return err
	}

	commitID, err := backend.Repos.ResolveRev(ctx, repo, "")
	if err != nil {
		return err
	}
	_ = commitID

	vfs, err := repoVFS(r.Context(), repoName, commitID)
	if err != nil {
		return err
	}

	pkgPath := path.Join("/", dir.RepoPrefix, strings.TrimPrefix(dir.ImportPath, dir.ProjectRoot))
	location, err := symbolLocation(r.Context(), vfs, commitID, spec, pkgPath)
	if err != nil {
		return err
	}
	if location == nil {
		return &errcode.HTTPErr{
			Status: http.StatusNotFound,
			Err:    errors.New("symbol not found"),
		}
	}

	uri, err := url.Parse(string(location.URI))
	if err != nil {
		return err
	}
	filePath := uri.Fragment
	dest := &url.URL{
		Path:     "/" + path.Join(string(repo.Name), "-/blob", filePath),
		Fragment: fmt.Sprintf("L%d:%d$references", location.Range.Start.Line+1, location.Range.Start.Character+1),
	}
	http.Redirect(w, r, dest.String(), http.StatusFound)
	return nil
}

type goSymbolSpec struct {
	Pkg      string
	Receiver *string
	Symbol   string
}

type invalidSymbolURLPathError struct {
	Path string
}

func (s *invalidSymbolURLPathError) Error() string {
	return "invalid symbol URL path: " + s.Path
}

func parseGoSymbolURLPath(path string) (*goSymbolSpec, error) {
	parts := strings.SplitN(strings.Trim(path, "/"), "/", 2)
	if len(parts) < 2 {
		return nil, &invalidSymbolURLPathError{Path: path}
	}
	mode := parts[0]
	symbolID := parts[1]

	if mode != "go" {
		return nil, &errcode.HTTPErr{
			Status: http.StatusNotFound,
			Err:    errors.New("invalid mode (only \"go\" is supported"),
		}
	}

	//                                                        def
	//                                                    vvvvvvvvvvvv
	// http://sourcegraph.com/go/github.com/gorilla/mux/-/Router/Match
	//                           ^^^^^^^^^^^^^^^^^^^^^^   ^^^^^^ ^^^^^
	//                                 importPath      receiver? symbolname
	parts = strings.SplitN(symbolID, "/-/", 2)
	if len(parts) < 2 {
		return nil, &invalidSymbolURLPathError{Path: path}
	}
	importPath := parts[0]
	def := parts[1]
	var symbolName string
	var receiver *string
	symbolComponents := strings.Split(def, "/")
	switch len(symbolComponents) {
	case 1:
		symbolName = symbolComponents[0]
	case 2:
		// This is a method call.
		receiver = &symbolComponents[0]
		symbolName = symbolComponents[1]
	default:
		return nil, errors.Errorf("invalid def %s (must have 1 or 2 path components)", def)
	}

	return &goSymbolSpec{
		Pkg:      importPath,
		Receiver: receiver,
		Symbol:   symbolName,
	}, nil
}

func symbolLocation(ctx context.Context, vfs ctxvfs.FileSystem, commitID api.CommitID, symbolSpec *goSymbolSpec, pkgPath string) (*lsp.Location, error) {
	bctx := buildContextFromVFS(ctx, vfs)

	fileSet := token.NewFileSet()
	pkg, err := parseFiles(fileSet, &bctx, symbolSpec.Pkg, pkgPath)
	if err != nil {
		return nil, err
	}

	pos := (func() *token.Pos {
		docPackage := doc.New(pkg, symbolSpec.Pkg, doc.AllDecls)
		for _, docConst := range docPackage.Consts {
			for _, spec := range docConst.Decl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, ident := range valueSpec.Names {
						if ident.Name == symbolSpec.Symbol {
							return &ident.NamePos
						}
					}
				}
			}
		}
		for _, docType := range docPackage.Types {
			if symbolSpec.Receiver != nil && docType.Name == *symbolSpec.Receiver {
				for _, method := range docType.Methods {
					if method.Name == symbolSpec.Symbol {
						return &method.Decl.Name.NamePos
					}
				}
			}
			for _, fun := range docType.Funcs {
				if fun.Name == symbolSpec.Symbol {
					return &fun.Decl.Name.NamePos
				}
			}
			for _, spec := range docType.Decl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == symbolSpec.Symbol {
					return &typeSpec.Name.NamePos
				}
			}
		}
		for _, docVar := range docPackage.Vars {
			for _, spec := range docVar.Decl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, ident := range valueSpec.Names {
						if ident.Name == symbolSpec.Symbol {
							return &ident.NamePos
						}
					}
				}
			}
		}
		for _, docFunc := range docPackage.Funcs {
			if docFunc.Name == symbolSpec.Symbol {
				return &docFunc.Decl.Name.NamePos
			}
		}
		return nil
	})()

	if pos == nil {
		return nil, nil
	}

	position := fileSet.Position(*pos)
	location := lsp.Location{
		URI: lsp.DocumentURI("https://" + symbolSpec.Pkg + "?" + string(commitID) + "#" + position.Filename),
		Range: lsp.Range{
			Start: lsp.Position{
				Line:      position.Line - 1,
				Character: position.Column - 1,
			},
			End: lsp.Position{
				Line:      position.Line - 1,
				Character: position.Column - 1,
			},
		},
	}

	return &location, nil
}

func buildContextFromVFS(ctx context.Context, vfs ctxvfs.FileSystem) build.Context {
	bctx := build.Default
	PrepareContext(&bctx, ctx, vfs)
	return bctx
}

func repoVFS(ctx context.Context, name api.RepoName, rev api.CommitID) (ctxvfs.FileSystem, error) {
	if strings.HasPrefix(string(name), "github.com/") {
		return vfsutil.NewGitHubRepoVFS(string(name), string(rev))
	}

	// Fall back to a full git clone for non-github.com repos.
	return nil, errors.Errorf("unable to fetch repo %s (only github.com repos are supported)", name)
}

func parseFiles(fset *token.FileSet, bctx *build.Context, importPath, srcDir string) (*ast.Package, error) {
	bpkg, err := bctx.ImportDir(srcDir, 0)
	if err != nil {
		return nil, err
	}

	pkg := &ast.Package{
		Files: map[string]*ast.File{},
	}
	var errs error
	for _, file := range append(bpkg.GoFiles, bpkg.TestGoFiles...) {
		if src, err := buildutil.ParseFile(fset, bctx, nil, buildutil.JoinPath(bctx, srcDir), file, 0); err == nil {
			pkg.Name = src.Name.Name
			pkg.Files[file] = src
		} else {
			errs = multierror.Append(errs, err)
		}
	}

	return pkg, errs
}

func PrepareContext(bctx *build.Context, ctx context.Context, vfs ctxvfs.FileSystem) {
	// HACK: in the all Context's methods below we are trying to convert path to virtual one (/foo/bar/..)
	// because some code may pass OS-specific arguments.
	// See golang.org/x/tools/go/buildutil/allpackages.go which uses `filepath` for example

	bctx.OpenFile = func(path string) (io.ReadCloser, error) {
		path = filepath.ToSlash(path)
		return vfs.Open(ctx, path)
	}
	bctx.IsDir = func(path string) bool {
		path = filepath.ToSlash(path)
		fi, err := vfs.Stat(ctx, path)
		return err == nil && fi.Mode().IsDir()
	}
	bctx.HasSubdir = func(root, dir string) (rel string, ok bool) {
		if !bctx.IsDir(dir) {
			return "", false
		}
		if !PathHasPrefix(dir, root) {
			return "", false
		}
		return PathTrimPrefix(dir, root), true
	}
	bctx.ReadDir = func(path string) ([]fs.FileInfo, error) {
		path = filepath.ToSlash(path)
		return vfs.ReadDir(ctx, path)
	}
	bctx.IsAbsPath = func(path string) bool {
		path = filepath.ToSlash(path)
		return IsAbs(path)
	}
	bctx.JoinPath = func(elem ...string) string {
		// convert all backslashes to slashes to avoid
		// weird paths like C:\mygopath\/src/github.com/...
		for i, el := range elem {
			elem[i] = filepath.ToSlash(el)
		}
		return path.Join(elem...)
	}
}

func trimFilePrefix(s string) string {
	return strings.TrimPrefix(s, "file://")
}

func normalizePath(s string) string {
	if isURI(s) {
		return UriToPath(lsp.DocumentURI(s))
	}
	s = filepath.ToSlash(s)
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	return s
}

// PathHasPrefix returns true if s is starts with the given prefix
func PathHasPrefix(s, prefix string) bool {
	s = normalizePath(s)
	prefix = normalizePath(prefix)
	if s == prefix {
		return true
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return s == prefix || strings.HasPrefix(s, prefix)
}

// PathTrimPrefix removes the prefix from s
func PathTrimPrefix(s, prefix string) string {
	s = normalizePath(s)
	prefix = normalizePath(prefix)
	if s == prefix {
		return ""
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return strings.TrimPrefix(s, prefix)
}

// PathEqual returns true if both a and b are equal
func PathEqual(a, b string) bool {
	return PathTrimPrefix(a, b) == ""
}

// IsVendorDir tells if the specified directory is a vendor directory.
func IsVendorDir(dir string) bool {
	return strings.HasPrefix(dir, "vendor/") || strings.Contains(dir, "/vendor/")
}

// IsURI tells if s denotes an URI
func IsURI(s lsp.DocumentURI) bool {
	return isURI(string(s))
}

func isURI(s string) bool {
	return strings.HasPrefix(s, "file://")
}

// PathToURI converts given absolute path to file URI
func PathToURI(path string) lsp.DocumentURI {
	path = filepath.ToSlash(path)
	parts := strings.SplitN(path, "/", 2)

	// If the first segment is a Windows drive letter, prefix with a slash and skip encoding
	head := parts[0]
	if head != "" {
		head = "/" + head
	}

	rest := ""
	if len(parts) > 1 {
		rest = "/" + parts[1]
	}

	return lsp.DocumentURI("file://" + head + rest)
}

// UriToPath converts given file URI to path
func UriToPath(uri lsp.DocumentURI) string {
	u, err := url.Parse(string(uri))
	if err != nil {
		return trimFilePrefix(string(uri))
	}
	return u.Path
}

var regDriveLetter = lazyregexp.New("^/[a-zA-Z]:")

// UriToRealPath converts the given file URI to the platform specific path
func UriToRealPath(uri lsp.DocumentURI) string {
	path := UriToPath(uri)

	if regDriveLetter.MatchString(path) {
		// remove the leading slash if it starts with a drive letter
		// and convert to back slashes
		path = filepath.FromSlash(path[1:])
	}

	return path
}

// IsAbs returns true if the given path is absolute
func IsAbs(path string) bool {
	// Windows implementation accepts path-like and filepath-like arguments
	return strings.HasPrefix(path, "/") || filepath.IsAbs(path)
}

// Panicf takes the return value of recover() and outputs data to the log with
// the stack trace appended. Arguments are handled in the manner of
// fmt.Printf. Arguments should format to a string which identifies what the
// panic code was doing. Returns a non-nil error if it recovered from a panic.
func Panicf(r interface{}, format string, v ...interface{}) error {
	if r != nil {
		// Same as net/http
		const size = 64 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		id := fmt.Sprintf(format, v...)
		log.Printf("panic serving %s: %v\n%s", id, r, string(buf))
		return errors.Errorf("unexpected panic: %v", r)
	}
	return nil
}
