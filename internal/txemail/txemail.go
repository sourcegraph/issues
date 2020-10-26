// Package txemail sends transactional emails.
package txemail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"

	"github.com/jordan-wright/email"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/txemail/txtypes"
)

// Message describes an email message to be sent.
type Message struct {
	FromName   string   // email "From" address proper name
	To         []string // email "To" recipients
	ReplyTo    *string  // optional "ReplyTo" address
	MessageID  *string  // optional "Message-ID" header
	References []string // optional "References" header list

	Template txtypes.Templates // unparsed subject/body templates
	Data     interface{}       // template data
}

// render returns the rendered message contents without sending email.
func render(message Message) (*email.Email, error) {
	m := email.Email{
		To:      message.To,
		From:    "Sourcegraph",
		Headers: make(textproto.MIMEHeader),
	}
	if message.ReplyTo != nil {
		m.ReplyTo = []string{*message.ReplyTo}
	}
	if message.MessageID != nil {
		m.Headers["Message-ID"] = []string{*message.MessageID}
	}
	if message.FromName != "" {
		m.From = message.FromName
	}
	if len(message.References) > 0 {
		// jordan-wright/email does not support lists, so we must build it ourself.
		var refsList string
		for _, ref := range message.References {
			if refsList != "" {
				refsList += " "
			}
			refsList += fmt.Sprintf("<%s>", ref)
		}
		m.Headers["References"] = []string{refsList}
	}

	parsed, err := ParseTemplate(message.Template)
	if err != nil {
		return nil, err
	}

	if err := renderTemplate(parsed, message.Data, &m); err != nil {
		return nil, err
	}

	return &m, nil
}

// Send sends a transactional email.
//
// Callers that do not live in the frontend should call api.InternalClient.SendEmail
// instead. TODO(slimsag): needs cleanup as part of upcoming configuration refactor.
func Send(ctx context.Context, message Message) error {
	if MockSend != nil {
		return MockSend(ctx, message)
	}
	if disableSilently {
		return nil
	}

	conf := conf.Get()
	if conf.EmailAddress == "" {
		return errors.New("no \"From\" email address configured (in email.address)")
	}
	if conf.EmailSmtp == nil {
		return errors.New("no SMTP server configured (in email.smtp)")
	}

	m, err := render(message)
	if err != nil {
		return err
	}
	m.From = conf.EmailAddress

	// Disable Mandrill features, because they make the emails look sketchy.
	if conf.EmailSmtp.Host == "smtp.mandrillapp.com" {
		// Disable click tracking ("noclicks" could be any string; the docs say that anything will disable click tracking except
		// those defined at
		// https://mandrill.zendesk.com/hc/en-us/articles/205582117-How-to-Use-SMTP-Headers-to-Customize-Your-Messages#enable-open-and-click-tracking).
		m.Headers["X-MC-Track"] = []string{"noclicks"}

		m.Headers["X-MC-AutoText"] = []string{"false"}
		m.Headers["X-MC-AutoHTML"] = []string{"false"}
		m.Headers["X-MC-ViewContentLink"] = []string{"false"}
	}

	var smtpAuth smtp.Auth
	switch conf.EmailSmtp.Authentication {
	case "none": // nothing to do
	case "PLAIN":
		smtpAuth = smtp.PlainAuth("", conf.EmailSmtp.Username, conf.EmailSmtp.Password, conf.EmailSmtp.Host)
	case "CRAM-MD5":
		smtpAuth = smtp.CRAMMD5Auth(conf.EmailSmtp.Username, conf.EmailSmtp.Password)
	default:
		return fmt.Errorf("invalid SMTP authentication type %q", conf.EmailSmtp.Authentication)
	}

	if conf.EmailSmtp.DisableTLS {
		return m.SendWithStartTLS(
			net.JoinHostPort(conf.EmailSmtp.Host, strconv.Itoa(conf.EmailSmtp.Port)),
			smtpAuth,
			&tls.Config{
				InsecureSkipVerify: true,
			},
		)
	}

	return m.Send(
		net.JoinHostPort(conf.EmailSmtp.Host, strconv.Itoa(conf.EmailSmtp.Port)),
		smtpAuth,
	)
}

// MockSend is used in tests to mock the Send func.
var MockSend func(ctx context.Context, message Message) error

var disableSilently bool

// DisableSilently prevents sending of emails, even if email sending is
// configured. Use it in tests to ensure that they do not send real emails.
func DisableSilently() {
	disableSilently = true
}
