package bounce

import (
	"testing"

	"github.com/lumen/relay/internal/domain"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		code, enh, msg string
		want           domain.BounceClass
	}{
		{"550", "5.1.1", "user unknown", domain.BounceHard},
		{"550", "", "mailbox unavailable / user unknown", domain.BounceHard},
		{"421", "4.2.1", "try again", domain.BounceSoft},
		{"550", "5.7.1", "blocked as spam", domain.BounceBlock},
		{"250", "", "ok", domain.BounceOK},
	}
	for _, c := range cases {
		got := Classify(c.code, c.enh, c.msg)
		if got != c.want {
			t.Fatalf("%s %s => %s want %s", c.code, c.msg, got, c.want)
		}
	}
}

func TestSoftThreshold(t *testing.T) {
	ev := domain.BounceEvent{Class: domain.BounceSoft}
	a := Next("subscribed", ev, 2)
	if !a.Suppress || a.Reason != "soft_bounce_threshold" {
		t.Fatalf("%+v", a)
	}
}

func TestHardIsolates(t *testing.T) {
	a := Next("subscribed", domain.BounceEvent{Class: domain.BounceHard}, 0)
	if !a.Suppress || a.ContactStatus != "suppressed" {
		t.Fatal(a)
	}
}
