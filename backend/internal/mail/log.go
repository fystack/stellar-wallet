package mail

import "go.uber.org/zap"

// LogMailer logs single-use invite links, so it is wired in only for a localhost frontend.
type LogMailer struct{}

func (LogMailer) Send(to string, message Message) error {
	zap.L().Info("email not sent (SMTP disabled, local log mailer)",
		zap.String("to", to), zap.String("subject", message.Subject), zap.String("link", message.ActionURL))
	return nil
}
