package provider

import "testing"

func TestPickSkipsOpen(t *testing.T) {
	b := NewBalancer([]Health{
		{Key: "gmail", Weight: 10, Score: 1, State: "open"},
		{Key: "outlook", Weight: 1, Score: 1, State: "closed"},
	})
	var gmail, outlook int
	for i := 0; i < 50; i++ {
		h, ok := b.Pick()
		if !ok {
			t.Fatal("none")
		}
		if h.Key == "gmail" {
			gmail++
		} else {
			outlook++
		}
	}
	if gmail != 0 || outlook != 50 {
		t.Fatalf("gmail=%d outlook=%d", gmail, outlook)
	}
}

func TestCircuitOpens(t *testing.T) {
	b := NewBalancer([]Health{{Key: "a", Weight: 1, Score: 1, State: "closed"}})
	for i := 0; i < 5; i++ {
		b.Report("a", false)
	}
	if b.Snapshot()[0].State != "open" {
		t.Fatal(b.Snapshot()[0])
	}
}
