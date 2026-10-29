// Package mailer defines the Mailer port used by features that need to send
// email.
//
// The port lives in pkg/ so that every feature that needs email shares a single
// definition. The concrete implementation (SMTP) lives in
// internal/<feature>/adapter/mailer_smtp.go.
package mailer

import "context"

// Mailer sends email. It is a port, not an implementation.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// Fake is a no-op Mailer for tests. It records every message it is asked to
// send so tests can assert on them.
type Fake struct {
	Messages []Message
}

// Message is a single recorded email.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Send records the message and returns nil.
func (f *Fake) Send(_ context.Context, to, subject, body string) error {
	f.Messages = append(f.Messages, Message{
		To:      to,
		Subject: subject,
		Body:    body,
	})

	return nil
}
