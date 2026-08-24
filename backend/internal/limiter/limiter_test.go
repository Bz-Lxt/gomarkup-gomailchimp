package limiter

import (
	"context"
	"testing"
	"time"
)

func TestMemoryBucketRate(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	b := NewMemory(func() time.Time { return now })
	ctx := context.Background()
	ok, _ := b.Allow(ctx, "k", 60, 1)
	if !ok {
		t.Fatal("first token")
	}
	ok, _ = b.Allow(ctx, "k", 60, 1)
	if ok {
		t.Fatal("burst exhausted")
	}
	now = now.Add(time.Second)
	ok, _ = b.Allow(ctx, "k", 60, 1)
	if !ok {
		t.Fatal("refilled after 1s at 60/min")
	}
}

func TestChannelIsolation(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	b := NewMemory(func() time.Time { return now })
	m := &Multi{AllowFn: func(ctx context.Context, key string, rate, burst float64) (bool, float64, error) {
		ok, left := b.Allow(ctx, key, rate, burst)
		return ok, left, nil
	}}
	ctx := context.Background()
	gmail := []Dim{{Key: "dom:gmail", RatePerMin: 60, Burst: 1}}
	outlook := []Dim{{Key: "dom:outlook", RatePerMin: 60, Burst: 60}}
	d1, _ := m.AllowAll(ctx, gmail)
	if !d1.Allowed {
		t.Fatal("gmail first")
	}
	d2, _ := m.AllowAll(ctx, gmail)
	if d2.Allowed {
		t.Fatal("gmail should block")
	}
	var outlookOK int
	for i := 0; i < 20; i++ {
		d, _ := m.AllowAll(ctx, outlook)
		if d.Allowed {
			outlookOK++
		}
	}
	if outlookOK < 19 {
		t.Fatalf("outlook blocked by gmail: got %d", outlookOK)
	}
}

func TestDomainClass(t *testing.T) {
	if DomainClass("a@gmail.com") != "gmail" {
		t.Fatal("gmail")
	}
	if DomainClass("b@outlook.com") != "outlook" {
		t.Fatal("outlook")
	}
	if DomainClass("c@shop.io") != "other" {
		t.Fatal("other")
	}
}
