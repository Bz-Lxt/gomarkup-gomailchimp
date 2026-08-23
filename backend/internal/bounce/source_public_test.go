package bounce_test

import (
	"context"
	"testing"

	"github.com/lumen/relay/internal/bounce"
	"github.com/lumen/relay/internal/domain"
)

func TestMockFeederPolledBatchRemainsStable(t *testing.T) {
	feeder := &bounce.MockFeeder{}
	permanent := []byte(`{
		"notificationType":"Bounce",
		"bounce":{
			"bounceType":"Permanent",
			"bouncedRecipients":[{
				"emailAddress":"first@example.com",
				"status":"5.1.1",
				"diagnosticCode":"mailbox unavailable"
			}]
		}
	}`)
	if err := feeder.InjectSES(permanent); err != nil {
		t.Fatalf("inject permanent bounce: %v", err)
	}

	firstBatch, err := feeder.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll first batch: %v", err)
	}
	if len(firstBatch) != 1 {
		t.Fatalf("first batch length = %d, want 1", len(firstBatch))
	}

	complaint := []byte(`{
		"notificationType":"Complaint",
		"complaint":{
			"complainedRecipients":[{"emailAddress":"second@example.com"}]
		}
	}`)
	if err := feeder.InjectSES(complaint); err != nil {
		t.Fatalf("inject complaint: %v", err)
	}

	if got := firstBatch[0]; got.Email != "first@example.com" || got.Class != domain.BounceHard || got.Enhanced != "5.1.1" {
		t.Fatalf("previously polled event changed after a later injection: %+v", got)
	}

	secondBatch, err := feeder.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll second batch: %v", err)
	}
	if len(secondBatch) != 1 || secondBatch[0].Email != "second@example.com" || secondBatch[0].Class != domain.BounceBlock {
		t.Fatalf("second batch = %+v, want complaint for second@example.com", secondBatch)
	}
}
