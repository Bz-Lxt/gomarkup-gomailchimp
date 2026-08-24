package pipeline_test

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/lumen/relay/internal/pipeline"
	"github.com/redis/go-redis/v9"
)

type claimStoreHook struct {
	mu         sync.Mutex
	claimed    bool
	exists     int
	bothExists chan struct{}
}

func newClaimStoreHook() *claimStoreHook {
	return &claimStoreHook{bothExists: make(chan struct{})}
}

func (h *claimStoreHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (h *claimStoreHook) ProcessHook(redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		switch c := cmd.(type) {
		case *redis.BoolCmd:
			h.mu.Lock()
			won := !h.claimed
			if won {
				h.claimed = true
			}
			h.mu.Unlock()
			c.SetVal(won)
			return nil
		case *redis.IntCmd:
			if cmd.Name() != "exists" {
				return fmt.Errorf("unexpected integer command %q", cmd.Name())
			}
			h.mu.Lock()
			h.exists++
			if h.exists == 2 {
				close(h.bothExists)
			}
			h.mu.Unlock()
			select {
			case <-h.bothExists:
				c.SetVal(0)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		case *redis.StatusCmd:
			h.mu.Lock()
			h.claimed = true
			h.mu.Unlock()
			c.SetVal("OK")
			return nil
		default:
			return fmt.Errorf("unexpected Redis command %q", cmd.Name())
		}
	}
}

func (h *claimStoreHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		return next(ctx, cmds)
	}
}

func TestClaimIdemConcurrentSingleWinner(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "unused:0"})
	rdb.AddHook(newClaimStoreHook())
	t.Cleanup(func() { _ = rdb.Close() })

	q := pipeline.NewQueue(rdb)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	start := make(chan struct{})
	results := make(chan bool, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			won, err := q.ClaimIdem(ctx, "same-message", time.Minute)
			results <- won
			errs <- err
		}()
	}
	close(start)

	winners := 0
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("claim failed: %v", err)
		}
		if <-results {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("same message claimed by %d workers; want exactly one", winners)
	}
}
