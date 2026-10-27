// Package mail sends transactional email over SMTP.
package mail

import (
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"stellar-wallet-backend/internal/config"
)

type SMTPMailer struct {
	config config.SMTPConfig
}

// New returns nil when SMTP is not configured, which disables invite emails.
func New(cfg config.SMTPConfig) *SMTPMailer {
	if cfg.Host == "" {
		return nil
	}
	return &SMTPMailer{config: cfg}
}

func (m *SMTPMailer) Send(to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("send mail: invalid recipient")
	}
	message := strings.Join([]string{
		"From: " + m.config.From,
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		strings.ReplaceAll(body, "\n", "\r\n"),
	}, "\r\n")
	var authentication smtp.Auth
	if m.config.Username != "" {
		authentication = smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	}
	sender, err := mail.ParseAddress(m.config.From)
	if err != nil {
		return fmt.Errorf("send mail: parse from address: %w", err)
	}
	address := net.JoinHostPort(m.config.Host, strconv.Itoa(m.config.Port))
	if err := smtp.SendMail(address, authentication, sender.Address, []string{to}, []byte(message)); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}
