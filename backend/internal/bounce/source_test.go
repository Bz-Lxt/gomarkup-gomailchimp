package bounce

import (
	"context"
	"testing"

	"github.com/lumen/relay/internal/domain"
)

func sesBounceJSON(email, bounceType, status, diag string) []byte {
	return []byte(`{"notificationType":"Bounce","bounce":{"bounceType":"` + bounceType +
		`","bouncedRecipients":[{"emailAddress":"` + email +
		`","status":"` + status + `","diagnosticCode":"` + diag + `"}]}}`)
}

func sesComplaintJSON(email string) []byte {
	return []byte(`{"notificationType":"Complaint","complaint":{"complainedRecipients":[{"emailAddress":"` + email + `"}]}}`)
}

// TestMockFeederBatchHandover reproduces the intermittent cross-batch data
// corruption: after a worker polls the first batch (permanent bounce for
// first@example.com) but before it is persisted, the feeder receives a
// complaint for second@example.com. The first batch's content must NOT be
// overwritten by the subsequent notification.
func TestMockFeederBatchHandover(t *testing.T) {
	feed := &MockFeeder{}

	// 1. Feeder receives a permanent bounce for first@example.com
	if err := feed.InjectSES(sesBounceJSON("first@example.com", "Permanent", "5.1.1", "user unknown")); err != nil {
		t.Fatal(err)
	}

	// 2. Worker polls and caches the first batch (not yet persisted)
	batch1, err := feed.Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(batch1) != 1 {
		t.Fatalf("batch1 len = %d, want 1", len(batch1))
	}
	if batch1[0].Email != "first@example.com" {
		t.Fatalf("batch1[0].email = %s, want first@example.com", batch1[0].Email)
	}
	if batch1[0].Class != domain.BounceHard {
		t.Fatalf("batch1[0].class = %s, want hard", batch1[0].Class)
	}

	// 3. Before batch1 is persisted, feeder receives complaint for second@example.com
	if err := feed.InjectSES(sesComplaintJSON("second@example.com")); err != nil {
		t.Fatal(err)
	}

	// 4. The first batch must be immutable — not overwritten by the second notification
	if batch1[0].Email != "first@example.com" {
		t.Fatalf("batch1[0].email corrupted: got %s, want first@example.com", batch1[0].Email)
	}
	if batch1[0].Class != domain.BounceHard {
		t.Fatalf("batch1[0].class corrupted: got %s, want hard", batch1[0].Class)
	}
	if batch1[0].Message != "user unknown" {
		t.Fatalf("batch1[0].message corrupted: got %s, want 'user unknown'", batch1[0].Message)
	}

	// 5. Second batch should contain the complaint independently
	batch2, err := feed.Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(batch2) != 1 {
		t.Fatalf("batch2 len = %d, want 1", len(batch2))
	}
	if batch2[0].Email != "second@example.com" {
		t.Fatalf("batch2[0].email = %s, want second@example.com", batch2[0].Email)
	}
	if batch2[0].Class != domain.BounceBlock {
		t.Fatalf("batch2[0].class = %s, want block", batch2[0].Class)
	}

	// 6. After batch2 is polled, batch1 must still be intact
	if batch1[0].Email != "first@example.com" {
		t.Fatalf("batch1[0].email re-corrupted: got %s", batch1[0].Email)
	}
}

// TestMockFeederMultiRecipientHandover verifies that a multi-recipient first
// batch is not partially overwritten when a second notification arrives.
func TestMockFeederMultiRecipientHandover(t *testing.T) {
	feed := &MockFeeder{}

	multi := []byte(`{"notificationType":"Bounce","bounce":{"bounceType":"Permanent","bouncedRecipients":[` +
		`{"emailAddress":"a@example.com","status":"5.1.1","diagnosticCode":"no such user"},` +
		`{"emailAddress":"b@example.com","status":"5.2.1","diagnosticCode":"mailbox full"}` +
		`]}}`)
	if err := feed.InjectSES(multi); err != nil {
		t.Fatal(err)
	}

	batch1, _ := feed.Poll(context.Background())
	if len(batch1) != 2 {
		t.Fatalf("batch1 len = %d, want 2", len(batch1))
	}

	// Second notification arrives while batch1 is still in-flight
	if err := feed.InjectSES(sesComplaintJSON("c@example.com")); err != nil {
		t.Fatal(err)
	}

	if batch1[0].Email != "a@example.com" || batch1[1].Email != "b@example.com" {
		t.Fatalf("batch1 corrupted: [%s, %s]", batch1[0].Email, batch1[1].Email)
	}
	if batch1[0].Class != domain.BounceHard || batch1[1].Class != domain.BounceHard {
		t.Fatalf("batch1 class corrupted: [%s, %s]", batch1[0].Class, batch1[1].Class)
	}
}
