package adapter

import (
	"context"
	"fmt"
	"net/smtp"
	"strconv"

	"github.com/rifkibyte/gin-boilerplate/internal/config"
)

// SMTPMailer sends emails over SMTP.
type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewSMTPMailer constructs an SMTPMailer from MailConfig.
func NewSMTPMailer(cfg config.MailConfig) *SMTPMailer {
	return &SMTPMailer{
		host:     cfg.Host,
		port:     cfg.Port,
		username: cfg.Username,
		password: cfg.Password,
		from:     cfg.From,
	}
}

func (m *SMTPMailer) Send(_ context.Context, to, subject, body string) error {
	addr := m.host + ":" + strconv.Itoa(m.port)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", m.from, to, subject, body)

	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}