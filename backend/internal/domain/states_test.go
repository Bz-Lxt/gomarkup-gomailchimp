package domain

import "testing"

func TestCampaignMachine(t *testing.T) {
	if !CanTransitCampaign("draft", "running") {
		t.Fatal("draft→running")
	}
	if CanTransitCampaign("completed", "running") {
		t.Fatal("completed locked")
	}
	if !CanTransitCampaign("running", "paused") {
		t.Fatal("pause")
	}
}

func TestRecipientMachine(t *testing.T) {
	if !CanTransitRecipient("pending", "queued") {
		t.Fatal()
	}
	if CanTransitRecipient("skipped", "sent") {
		t.Fatal("skipped terminal")
	}
}
