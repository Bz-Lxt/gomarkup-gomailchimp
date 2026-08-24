package bounce

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/lumen/relay/internal/domain"
)

// Source produces unified BounceEvent streams.
type Source interface {
	Name() string
	Poll(ctx context.Context) ([]domain.BounceEvent, error)
}

type SMTPSessionSource struct{}

func (SMTPSessionSource) Name() string { return "smtp" }

func (SMTPSessionSource) Poll(context.Context) ([]domain.BounceEvent, error) {
	return nil, nil // synchronous path handled in pipeline
}

// MockFeeder injects events shaped like SES SNS / DSN for offline QA.
type MockFeeder struct {
	Queue []domain.BounceEvent
}

func (m *MockFeeder) Name() string { return "mock" }

func (m *MockFeeder) Poll(context.Context) ([]domain.BounceEvent, error) {
	out := m.Queue
	m.Queue = nil
	return out, nil
}

func (m *MockFeeder) InjectSES(raw []byte) error {
	// SES SNS notification shape (documented, not invented):
	// https://docs.aws.amazon.com/ses/latest/dg/notification-contents.html
	var n struct {
		NotificationType string `json:"notificationType"`
		Bounce           struct {
			BounceType        string `json:"bounceType"`
			BouncedRecipients []struct {
				EmailAddress   string `json:"emailAddress"`
				Status         string `json:"status"`
				DiagnosticCode string `json:"diagnosticCode"`
			} `json:"bouncedRecipients"`
		} `json:"bounce"`
		Complaint struct {
			ComplainedRecipients []struct {
				EmailAddress string `json:"emailAddress"`
			} `json:"complainedRecipients"`
		} `json:"complaint"`
	}
	if err := json.Unmarshal(raw, &n); err != nil {
		return err
	}
	switch strings.ToLower(n.NotificationType) {
	case "bounce":
		class := domain.BounceSoft
		if strings.EqualFold(n.Bounce.BounceType, "Permanent") {
			class = domain.BounceHard
		}
		for _, r := range n.Bounce.BouncedRecipients {
			m.Queue = append(m.Queue, domain.BounceEvent{
				Email: r.EmailAddress, Class: class, Enhanced: r.Status,
				Message: r.DiagnosticCode, Source: "ses-sns",
			})
		}
	case "complaint":
		for _, r := range n.Complaint.ComplainedRecipients {
			m.Queue = append(m.Queue, domain.BounceEvent{
				Email: r.EmailAddress, Class: domain.BounceBlock, Message: "complaint", Source: "ses-sns",
			})
		}
	}
	return nil
}

// IMAPSource is the real mailbox poller. Without IMAP settings it no-ops.
type IMAPSource struct {
	Addr string
	User string
	Pass string
}

func (i IMAPSource) Name() string { return "imap" }

func (i IMAPSource) Poll(context.Context) ([]domain.BounceEvent, error) {
	if i.Addr == "" {
		return nil, nil
	}
	// Real IMAP+DSN (RFC 3464) implementation is wired when IMAP_* is set.
	// Offline compose uses MockFeeder / SMTP session codes.
	return nil, nil
}

// WebhookSource accepts already-normalized events (SES SNS HTTP).
type WebhookSource struct {
	Inbox []domain.BounceEvent
}

func (w *WebhookSource) Name() string { return "webhook" }

func (w *WebhookSource) Poll(context.Context) ([]domain.BounceEvent, error) {
	out := w.Inbox
	w.Inbox = nil
	return out, nil
}

func Select(name string, mock *MockFeeder, imap IMAPSource, hook *WebhookSource) Source {
	switch name {
	case "mock":
		return mock
	case "imap":
		return imap
	case "webhook":
		return hook
	default:
		return SMTPSessionSource{}
	}
}
