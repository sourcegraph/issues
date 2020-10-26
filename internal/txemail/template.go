package txemail

import (
	"bytes"
	"html"
	htmltemplate "html/template"
	"io"
	"strings"
	texttemplate "text/template"

	"github.com/jordan-wright/email"
	"github.com/microcosm-cc/bluemonday"
	gfm "github.com/shurcooL/github_flavored_markdown"
	"github.com/sourcegraph/sourcegraph/internal/txemail/txtypes"
)

// MustParseTemplate calls ParseTemplate and panics if an error is returned.
// It is intended to be called in a package init func.
func MustParseTemplate(input txtypes.Templates) txtypes.ParsedTemplates {
	pt, err := ParseTemplate(input)
	if err != nil {
		panic("MustParseTemplate: " + err.Error())
	}
	return *pt
}

// MustValidate panics if the templates are unparsable, otherwise it returns
// them unmodified.
func MustValidate(input txtypes.Templates) txtypes.Templates {
	MustParseTemplate(input)
	return input
}

// ParseTemplate is a helper func for parsing the text/template and html/template
// templates together. In the future it will also provide common template funcs
// and a common footer.
func ParseTemplate(input txtypes.Templates) (*txtypes.ParsedTemplates, error) {
	st, err := texttemplate.New("").Funcs(textFuncMap).Parse(strings.TrimSpace(input.Subject))
	if err != nil {
		return nil, err
	}

	tt, err := texttemplate.New("").Funcs(textFuncMap).Parse(strings.TrimSpace(input.Text))
	if err != nil {
		return nil, err
	}

	ht, err := htmltemplate.New("").Funcs(htmlFuncMap).Parse(strings.TrimSpace(input.HTML))
	if err != nil {
		return nil, err
	}

	return &txtypes.ParsedTemplates{Subj: st, Text: tt, Html: ht}, nil
}

func renderTemplate(t *txtypes.ParsedTemplates, data interface{}, m *email.Email) error {
	render := func(tmpl interface {
		Execute(io.Writer, interface{}) error
	}) ([]byte, error) {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	subject, err := render(t.Subj)
	if err != nil {
		return err
	}
	m.Subject = string(subject)

	m.Text, err = render(t.Text)
	if err != nil {
		return err
	}

	m.HTML, err = render(t.Html)
	return err
}

var (
	textFuncMap = map[string]interface{}{
		// Removes HTML tags (which are valid Markdown) from the source, for display in a text-only
		// setting.
		"markdownToText": func(markdownSource string) string {
			p := bluemonday.StrictPolicy()
			return html.UnescapeString(p.Sanitize(markdownSource))
		},
	}

	htmlFuncMap = map[string]interface{}{
		// Renders Markdown for display in an HTML email.
		"markdownToSafeHTML": func(markdownSource string) htmltemplate.HTML {
			unsafeHTML := gfm.Markdown([]byte(markdownSource))

			// The recommended policy at https://github.com/russross/blackfriday#extensions
			p := bluemonday.UGCPolicy()
			return htmltemplate.HTML(p.SanitizeBytes(unsafeHTML))
		},
	}
)
