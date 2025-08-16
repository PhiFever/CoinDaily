package main

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

type EmailConfig struct {
	SMTPServer string
	SMTPPort   int
	Username   string
	Password   string
	To         string
}

type EmailSender struct {
	config EmailConfig
}

func NewEmailSender(config EmailConfig) *EmailSender {
	return &EmailSender{
		config: config,
	}
}

func (e *EmailSender) SendReport(subject string, htmlContent string) error {
	from := e.config.Username
	to := []string{e.config.To}
	
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(to, ",")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=utf-8"
	headers["Date"] = time.Now().Format(time.RFC1123Z)

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + htmlContent

	auth := smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.SMTPServer)
	addr := fmt.Sprintf("%s:%d", e.config.SMTPServer, e.config.SMTPPort)
	
	err := smtp.SendMail(addr, auth, from, to, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	return nil
}