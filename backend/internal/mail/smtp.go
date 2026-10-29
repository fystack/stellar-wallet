// Package mail sends transactional email over SMTP.
package mail

import (
	"crypto/tls"
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

const sendTimeout = 30 * time.Second

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
	if err := m.deliver(authentication, sender.Address, to, []byte(message)); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}

// deliver is smtp.SendMail with a deadline, which SendMail lacks.
func (m *SMTPMailer) deliver(authentication smtp.Auth, from, to string, message []byte) error {
	address := net.JoinHostPort(m.config.Host, strconv.Itoa(m.config.Port))
	connection, err := net.DialTimeout("tcp", address, sendTimeout)
	if err != nil {
		return err
	}
	if err := connection.SetDeadline(time.Now().Add(sendTimeout)); err != nil {
		connection.Close()
		return err
	}
	client, err := smtp.NewClient(connection, m.config.Host)
	if err != nil {
		connection.Close()
		return err
	}
	defer client.Close()
	if supported, _ := client.Extension("STARTTLS"); supported {
		if err := client.StartTLS(&tls.Config{ServerName: m.config.Host}); err != nil {
			return err
		}
	}
	if authentication != nil {
		if err := client.Auth(authentication); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
