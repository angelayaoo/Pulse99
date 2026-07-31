package notify

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type EmailNotifier struct {
	host        string
	port        int
	username    string
	password    string
	fromAddress string
	toAddresses []string
}

func NewEmail(host string, port int, username, password, from string, to []string) *EmailNotifier {
	return &EmailNotifier{
		host:        host,
		port:        port,
		username:    username,
		password:    password,
		fromAddress: from,
		toAddresses: to,
	}
}

func (e *EmailNotifier) Name() string { return "email" }

func (e *EmailNotifier) SendCritical(p Payload) error {
	if e.host == "" || len(e.toAddresses) == 0 {
		return nil
	}
	subject := fmt.Sprintf("CRITICAL: %s is DOWN -- Pulse99", p.Name)
	body := fmt.Sprintf(
		"Pulse99 -- Critical Alert\n"+
			"========================\n\n"+
			"[ Target ]     %s\n"+
			"[ URL ]        %s\n"+
			"[ Status ]     DOWN\n"+
			"[ Failures ]   %d consecutive\n"+
			"[ Cause ]      %s\n"+
			"[ Timestamp ]  %s UTC\n\n"+
			"-- Pulse99 Uptime Monitor",
		p.Name, p.URL, p.Failures, p.Reason, p.Timestamp.UTC().Format(time.RFC3339),
	)
	return e.send(subject, body)
}

func (e *EmailNotifier) SendRecovery(p Payload) error {
	if e.host == "" || len(e.toAddresses) == 0 {
		return nil
	}
	subject := fmt.Sprintf("RECOVERED: %s is ONLINE -- Pulse99", p.Name)
	body := fmt.Sprintf(
		"Pulse99 -- Recovery Alert\n"+
			"========================\n\n"+
			"[ Target ]     %s\n"+
			"[ URL ]        %s\n"+
			"[ Status ]     ONLINE\n"+
			"[ Latency ]    %dms\n"+
			"[ Timestamp ]  %s UTC\n\n"+
			"-- Pulse99 Uptime Monitor",
		p.Name, p.URL, p.LatencyMs, p.Timestamp.UTC().Format(time.RFC3339),
	)
	return e.send(subject, body)
}

func (e *EmailNotifier) send(subject, body string) error {
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		e.fromAddress,
		strings.Join(e.toAddresses, ", "),
		subject,
		body,
	)

	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}

	client, err := smtp.NewClient(conn, e.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Quit()

	tlsConfig := &tls.Config{ServerName: e.host}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}

	auth := smtp.PlainAuth("", e.username, e.password, e.host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	if err := client.Mail(e.fromAddress); err != nil {
		return fmt.Errorf("mail: %w", err)
	}
	for _, to := range e.toAddresses {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("rcpt: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	_, err = w.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	return nil
}
