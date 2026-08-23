package pipeline

import (
	"errors"
	"math"
	"time"

	"github.com/lumen/relay/internal/domain"
	"github.com/lumen/relay/internal/provider"
)

const MaxAttempts = 3

// NarrowRetry: only 4xx / network / Transient. Never retry auth or 5xx address errors.
func ShouldRetry(res provider.Result, err error, attempt int) (bool, time.Duration) {
	if attempt >= MaxAttempts {
		return false, 0
	}
	if res.AuthFailed {
		return false, 0
	}
	if res.Accepted {
		return false, 0
	}
	if res.Transient {
		return true, backoff(attempt)
	}
	if errors.Is(err, domain.ErrTransient) && res.Code == "" {
		return true, backoff(attempt)
	}
	return false, 0
}

func backoff(attempt int) time.Duration {
	// 1s, 2s, 4s
	sec := math.Pow(2, float64(attempt-1))
	if sec < 1 {
		sec = 1
	}
	return time.Duration(sec) * time.Second
}
