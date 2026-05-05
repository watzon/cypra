// Package email sends transactional email through pluggable backends.
package email

//revive:disable:exported

import (
	"context"
	"fmt"
	"io"
)

type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

type Sender interface {
	Send(ctx context.Context, message Message) error
}

type TerminalSender struct{ Writer io.Writer }

func (s TerminalSender) Send(_ context.Context, message Message) error {
	_, err := fmt.Fprintf(s.Writer, "To: %s\nSubject: %s\n\n%s\n", message.To, message.Subject, message.Text)
	return err
}
