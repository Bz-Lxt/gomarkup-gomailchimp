package provider

import (
	"sync"
)

// Balancer picks a healthy channel by weight * health, skipping open circuits.
type Balancer struct {
	mu       sync.RWMutex
	channels []Health
	cursor   int
}

func NewBalancer(chs []Health) *Balancer {
	return &Balancer{channels: chs}
}

func (b *Balancer) Snapshot() []Health {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]Health, len(b.channels))
	copy(out, b.channels)
	return out
}

func (b *Balancer) Pick() (Health, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	total := 0.0
	for _, c := range b.channels {
		if c.State == "open" {
			continue
		}
		w := float64(c.Weight)
		if w <= 0 {
			w = 1
		}
		h := c.Score
		if h <= 0 {
			h = 0.05
		}
		if c.State == "half" {
			h *= 0.2
		}
		total += w * h
	}
	if total <= 0 {
		return Health{}, false
	}
	// weighted round-robin via cursor
	b.cursor++
	target := float64(b.cursor % 1000)
	acc := 0.0
	for _, c := range b.channels {
		if c.State == "open" {
			continue
		}
		w := float64(c.Weight)
		if w <= 0 {
			w = 1
		}
		h := c.Score
		if h <= 0 {
			h = 0.05
		}
		if c.State == "half" {
			h *= 0.2
		}
		acc += (w * h) / total * 1000
		if target <= acc {
			return c, true
		}
	}
	for _, c := range b.channels {
		if c.State != "open" {
			return c, true
		}
	}
	return Health{}, false
}

func (b *Balancer) Report(key string, success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.channels {
		if b.channels[i].Key != key {
			continue
		}
		if success {
			b.channels[i].FailStreak = 0
			if b.channels[i].Score < 1 {
				b.channels[i].Score += 0.1
				if b.channels[i].Score > 1 {
					b.channels[i].Score = 1
				}
			}
			b.channels[i].State = "closed"
			return
		}
		b.channels[i].FailStreak++
		b.channels[i].Score *= 0.7
		if b.channels[i].FailStreak >= 5 {
			b.channels[i].State = "open"
		} else if b.channels[i].FailStreak >= 3 {
			b.channels[i].State = "half"
		}
		return
	}
}
