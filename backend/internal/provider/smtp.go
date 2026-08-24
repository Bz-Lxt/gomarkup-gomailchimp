package provider

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type SMTPSender struct {
	Host     string
	Port     int
	User     string
	Pass     string
	StartTLS bool
	label    string
}

func NewSMTP(host string, port int, user, pass string, startTLS bool, label string) *SMTPSender {
	if label == "" {
		label = "smtp"
	}
	return &SMTPSender{Host: host, Port: port, User: user, Pass: pass, StartTLS: startTLS, label: label}
}

func (s *SMTPSender) Name() string { return s.label }

func (s *SMTPSender) Send(ctx context.Context, m Mail) (Result, error) {
	start := time.Now()
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	from := m.FromEmail
	msg := buildRFC822(m)

	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Pass, s.Host)
	}

	d := net.Dialer{Timeout: 8 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return Result{Provider: s.label, Transient: true, Message: err.Error(), Latency: time.Since(start)}, err
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return Result{Provider: s.label, Transient: true, Message: err.Error(), Latency: time.Since(start)}, err
	}
	defer c.Close()

	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return Result{Provider: s.label, AuthFailed: true, Code: "535", Message: err.Error(), Latency: time.Since(start)}, err
		}
	}
	if err := c.Mail(from); err != nil {
		return classifySMTP(s.label, err, start), err
	}
	if err := c.Rcpt(m.To); err != nil {
		return classifySMTP(s.label, err, start), err
	}
	w, err := c.Data()
	if err != nil {
		return classifySMTP(s.label, err, start), err
	}
	if _, err := w.Write(msg); err != nil {
		return classifySMTP(s.label, err, start), err
	}
	if err := w.Close(); err != nil {
		return classifySMTP(s.label, err, start), err
	}
	_ = c.Quit()
	return Result{
		Provider:  s.label,
		MessageID: m.MessageID,
		Accepted:  true,
		Code:      "250",
		Message:   "ok",
		Latency:   time.Since(start),
	}, nil
}

func classifySMTP(label string, err error, start time.Time) Result {
	msg := err.Error()
	code := extractCode(msg)
	r := Result{Provider: label, Code: code, Message: msg, Latency: time.Since(start)}
	if strings.HasPrefix(code, "4") {
		r.Transient = true
	}
	if code == "535" || strings.Contains(strings.ToLower(msg), "auth") {
		r.AuthFailed = true
	}
	return r
}

func extractCode(msg string) string {
	for i := 0; i+2 < len(msg); i++ {
		if msg[i] >= '2' && msg[i] <= '5' && msg[i+1] >= '0' && msg[i+1] <= '9' && msg[i+2] >= '0' && msg[i+2] <= '9' {
			return msg[i : i+3]
		}
	}
	return ""
}

func buildRFC822(m Mail) []byte {
	var b strings.Builder
	b.WriteString("From: " + encodeFrom(m.FromName, m.FromEmail) + "\r\n")
	b.WriteString("To: " + m.To + "\r\n")
	if m.ReplyTo != "" {
		b.WriteString("Reply-To: " + m.ReplyTo + "\r\n")
	}
	b.WriteString("Subject: " + m.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Message-ID: <" + m.MessageID + ">\r\n")
	if m.UnsubURL != "" {
		b.WriteString("List-Unsubscribe: <" + m.UnsubURL + ">")
		if m.UnsubMailto != "" {
			b.WriteString(", <mailto:" + m.UnsubMailto + ">")
		}
		b.WriteString("\r\n")
		b.WriteString("List-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n")
	}
	for k, v := range m.Headers {
		b.WriteString(k + ": " + v + "\r\n")
	}
	bound := "lumen-" + m.MessageID
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + bound + "\"\r\n\r\n")
	b.WriteString("--" + bound + "\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(m.Text + "\r\n")
	b.WriteString("--" + bound + "\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n")
	b.WriteString(m.HTML + "\r\n")
	b.WriteString("--" + bound + "--\r\n")
	return []byte(b.String())
}

func encodeFrom(name, email string) string {
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", name, email)
}
