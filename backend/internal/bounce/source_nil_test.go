package bounce_test

import (
	"context"
	"testing"

	"github.com/lumen/relay/internal/bounce"
)

func TestSelectSMTPSafelyPolls(t *testing.T) {
	source := bounce.Select("smtp", &bounce.MockFeeder{}, bounce.IMAPSource{}, &bounce.WebhookSource{})
	events, err := source.Poll(context.Background())
	if err != nil {
		t.Fatalf("SMTP bounce poll returned an error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("SMTP bounce poll returned %d events, want none", len(events))
	}
}
