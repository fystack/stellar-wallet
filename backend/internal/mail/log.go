package mail

import "go.uber.org/zap"

// LogMailer logs single-use invite links, so it is wired in only for a localhost frontend.
type LogMailer struct{}

func (LogMailer) Send(to, subject, body string) error {
	zap.L().Info("email not sent (SMTP disabled, local log mailer)",
		zap.String("to", to), zap.String("subject", subject), zap.String("body", body))
	return nil
}
