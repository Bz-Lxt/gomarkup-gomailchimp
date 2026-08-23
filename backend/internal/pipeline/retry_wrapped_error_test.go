package pipeline_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/lumen/relay/internal/domain"
	"github.com/lumen/relay/internal/pipeline"
	"github.com/lumen/relay/internal/provider"
)

func TestShouldRetryWrappedTransientError(t *testing.T) {
	err := fmt.Errorf("send via primary: %w", domain.ErrTransient)

	retry, delay := pipeline.ShouldRetry(provider.Result{}, err, 1)
	if !retry {
		t.Fatal("wrapped transient provider error without a response code was not retried")
	}
	if delay != time.Second {
		t.Fatalf("first retry delay = %v, want %v", delay, time.Second)
	}
}
