package provider

import (
	"context"
	"strings"
	"sync"
	"time"
)

// MockSender is an in-memory sink for pipeline / rate-limit benchmarks.
// Real SMTP path remains wired via SMTPSender.
type MockSender struct {
	mu      sync.Mutex
	Sent    []Mail
	FailFor map[string]string // email suffix or exact → smtp code
}

func NewMock() *MockSender {
	return &MockSender{FailFor: map[string]string{}}
}

func (s *MockSender) Name() string { return "mock" }

func (s *MockSender) Send(_ context.Context, m Mail) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if code := s.matchFail(m.To); code != "" {
		r := Result{Provider: "mock", Code: code, Message: "mock reject " + code, Accepted: false}
		if strings.HasPrefix(code, "4") {
			r.Transient = true
		}
		if code == "535" {
			r.AuthFailed = true
		}
		return r, errorsText(code)
	}
	s.Sent = append(s.Sent, m)
	return Result{Provider: "mock", MessageID: m.MessageID, Accepted: true, Code: "250", Message: "ok", Latency: time.Millisecond}, nil
}

func (s *MockSender) matchFail(to string) string {
	if c, ok := s.FailFor[to]; ok {
		return c
	}
	for k, c := range s.FailFor {
		if strings.HasSuffix(to, k) {
			return c
		}
	}
	return ""
}

func (s *MockSender) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Sent)
}

type errStr string

func (e errStr) Error() string { return string(e) }

func errorsText(code string) error { return errStr("smtp " + code) }
