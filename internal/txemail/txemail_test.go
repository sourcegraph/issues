package txemail

import (
	"net/textproto"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jordan-wright/email"
	"github.com/sourcegraph/sourcegraph/internal/txemail/txtypes"
)

func TestRender(t *testing.T) {
	replyTo := "admin@sourcegraph.com"
	messageID := "1"

	msg := Message{
		FromName:   "foo",
		To:         []string{"bar1@sourcegraph.com", "bar2@sourcegraph.com"},
		ReplyTo:    &replyTo,
		MessageID:  &messageID,
		References: []string{"ref1", "ref2"},
		Template: txtypes.Templates{
			Subject: `{{.A}} subject {{.B}}`,
			Text: `
	{{.A}} text body {{.B}}
	`,
			HTML: `
	{{.A}} html body <span class="{{.B}}" />
	`,
		},
		Data: struct {
			A string
			B string
		}{
			A: "a",
			B: `<b>`,
		},
	}

	got, err := render(msg)
	if err != nil {
		t.Fatal(err)
	}

	want := &email.Email{
		ReplyTo: []string{replyTo},
		From:    "foo",
		To:      []string{"bar1@sourcegraph.com", "bar2@sourcegraph.com"},
		Subject: "a subject <b>",
		Text:    []byte("a text body <b>"),
		HTML:    []byte(`a html body <span class="&lt;b&gt;" />`),
		Headers: textproto.MIMEHeader{
			"Message-ID": []string{messageID},
			"References": []string{"<ref1> <ref2>"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}
