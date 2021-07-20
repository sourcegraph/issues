package ui

import (
	"bytes"
	"crypto/md5"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/ui/assets"
)

//go:embed app.html
var appHTML string

//go:embed error.html
var errorHTML string

// TODO(slimsag): tests for everything in this file

var (
	versionCacheMu sync.RWMutex
	versionCache   = make(map[string]string)

	_, noAssetVersionString = os.LookupEnv("WEBPACK_DEV_SERVER")
)

// Functions that are exposed to templates.
var funcMap = template.FuncMap{
	"version": func(fp string) (string, error) {
		if noAssetVersionString {
			return "", nil
		}

		// Check the cache for the version.
		versionCacheMu.RLock()
		version, ok := versionCache[fp]
		versionCacheMu.RUnlock()
		if ok {
			return version, nil
		}

		// Read file contents and calculate MD5 sum to represent version.
		f, err := assets.Assets.Open(fp)
		if err != nil {
			return "", err
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			return "", err
		}
		version = fmt.Sprintf("%x", md5.Sum(data))

		// Update cache.
		versionCacheMu.Lock()
		versionCache[fp] = version
		versionCacheMu.Unlock()
		return version, nil
	},
}

var (
	loadTemplateMu    sync.RWMutex
	loadTemplateCache = map[string]*template.Template{}
)

// loadTemplate loads the template with the given path. Also loaded along
// with that template is any templates under the shared/ directory.
func loadTemplate(path string) (*template.Template, error) {
	// Check the cache, first.
	loadTemplateMu.RLock()
	tmpl, ok := loadTemplateCache[path]
	loadTemplateMu.RUnlock()
	if ok && !env.InsecureDev {
		return tmpl, nil
	}

	tmpl, err := doLoadTemplate(path)
	if err != nil {
		return nil, err
	}

	// Update cache.
	loadTemplateMu.Lock()
	loadTemplateCache[path] = tmpl
	loadTemplateMu.Unlock()
	return tmpl, nil
}

// doLoadTemplate should only be called by loadTemplate.
func doLoadTemplate(path string) (*template.Template, error) {
	// Read the file.
	var data string
	switch path {
	case "app.html":
		data = appHTML
	case "error.html":
		data = errorHTML
	default:
		return nil, errors.Errorf("invalid template path %q", path)
	}
	tmpl, err := template.New(path).Funcs(funcMap).Parse(data)
	if err != nil {
		return nil, errors.Errorf("ui: failed to parse template %q: %v", path, err)
	}
	return tmpl, nil
}

// renderTemplate renders the template with the given name. The template name
// is its file name, relative to the template directory.
//
// The given data is accessible in the template via $.Foobar
func renderTemplate(w http.ResponseWriter, name string, data interface{}) error {
	root, err := loadTemplate(name)
	if err != nil {
		return err
	}

	// Write to a buffer to avoid a partially written response going to w
	// when an error would occur. Otherwise, our error page template rendering
	// will be corrupted.
	var buf bytes.Buffer
	if err := root.Execute(&buf, data); err != nil {
		return err
	}
	_, err = buf.WriteTo(w)
	return err
}
