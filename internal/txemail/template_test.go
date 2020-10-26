package txemail

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jordan-wright/email"
	"github.com/sourcegraph/sourcegraph/internal/txemail/txtypes"
)

func TestParseTemplate(t *testing.T) {
	pt, err := ParseTemplate(txtypes.Templates{
		Subject: `{{.A}} subject {{.B}}`,
		Text: `
{{.A}} text body {{.B}}
`,
		HTML: `
{{.A}} html body <span class="{{.B}}" />
`,
	})
	if err != nil {
		t.Fatal(err)
	}

	var m email.Email
	if err := renderTemplate(pt, struct {
		A string
		B string
	}{
		A: "a",
		B: `<b>`,
	}, &m); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(m.Subject, `a subject <b>`); diff != "" {
		t.Fatalf("(-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(string(m.Text), `a text body <b>`); diff != "" {
		t.Fatalf("(-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(string(m.HTML), `a html body <span class="&lt;b&gt;" />`); diff != "" {
		t.Fatalf("(-want +got):\n%s", diff)
	}
}
