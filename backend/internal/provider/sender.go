package provider

import (
	"context"
	"time"
)

type Mail struct {
	MessageID   string
	FromName    string
	FromEmail   string
	ReplyTo     string
	To          string
	Subject     string
	HTML        string
	Text        string
	Headers     map[string]string
	ChannelKey  string
	UnsubMailto string
	UnsubURL    string
}

type Result struct {
	Provider   string            `json:"provider"`
	MessageID  string            `json:"message_id"`
	Accepted   bool              `json:"accepted"`
	Code       string            `json:"code"`
	Enhanced   string            `json:"enhanced"`
	Message    string            `json:"message"`
	Transient  bool              `json:"transient"`
	AuthFailed bool              `json:"auth_failed"`
	Latency    time.Duration     `json:"-"`
	Raw        map[string]any    `json:"raw,omitempty"`
}

type Sender interface {
	Name() string
	Send(ctx context.Context, m Mail) (Result, error)
}

type Health struct {
	Key        string
	Weight     int
	Score      float64
	State      string // closed=healthy, open=tripped, half=probe
	FailStreak int
}
