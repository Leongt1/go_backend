package email

import (
	"context"
	"fmt"
	"net/smtp"
)

type SMTPProvider struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPProvider(host, port, username, password, from string) *SMTPProvider {
	return &SMTPProvider{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPProvider) Send(_ context.Context, input SendEmailInput) error {
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.from, input.To, input.Subject, input.HTML,
	)

	addr := s.host + ":" + s.port
	if err := smtp.SendMail(addr, auth, s.from, []string{input.To}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp: failed to send email: %w", err)
	}

	return nil
}
