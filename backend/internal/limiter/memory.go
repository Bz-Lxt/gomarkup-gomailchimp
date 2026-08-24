package limiter

import (
	"context"
	"sync"
	"time"
)

// MemoryBucket is the single-process control implementation.
// Production must use RedisBucket (Lua + Redis TIME).
type MemoryBucket struct {
	mu      sync.Mutex
	tokens  map[string]float64
	updated map[string]time.Time
	now     func() time.Time
}

func NewMemory(now func() time.Time) *MemoryBucket {
	if now == nil {
		now = time.Now
	}
	return &MemoryBucket{
		tokens:  map[string]float64{},
		updated: map[string]time.Time{},
		now:     now,
	}
}

func (b *MemoryBucket) Allow(_ context.Context, key string, ratePerMin float64, burst float64) (bool, float64) {
	if ratePerMin <= 0 {
		return true, burst
	}
	if burst < 1 {
		burst = 1
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	tok, ok := b.tokens[key]
	if !ok {
		tok = burst
	} else {
		elapsed := now.Sub(b.updated[key]).Seconds()
		tok += elapsed * (ratePerMin / 60.0)
		if tok > burst {
			tok = burst
		}
	}
	if tok < 1 {
		b.tokens[key] = tok
		b.updated[key] = now
		return false, tok
	}
	tok--
	b.tokens[key] = tok
	b.updated[key] = now
	return true, tok
}
