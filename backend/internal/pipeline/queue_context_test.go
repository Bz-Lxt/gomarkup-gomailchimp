package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/lumen/relay/internal/domain"
	"github.com/lumen/relay/internal/pipeline"
	"github.com/redis/go-redis/v9"
)

type promotionHook struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once

	mu      sync.Mutex
	delayed string
	queued  string
}

func (h *promotionHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (h *promotionHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		switch cmd.Name() {
		case "zrangebyscore":
			h.once.Do(func() { close(h.started) })
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-h.release:
			}
			h.mu.Lock()
			defer h.mu.Unlock()
			vals := []string(nil)
			if h.delayed != "" {
				vals = append(vals, h.delayed)
			}
			cmd.(*redis.StringSliceCmd).SetVal(vals)
			return nil
		case "zrem":
			if err := ctx.Err(); err != nil {
				return err
			}
			h.mu.Lock()
			h.delayed = ""
			h.mu.Unlock()
			cmd.(*redis.IntCmd).SetVal(1)
			return nil
		case "lpush":
			if err := ctx.Err(); err != nil {
				return err
			}
			h.mu.Lock()
			h.queued = cmd.Args()[2].(string)
			h.mu.Unlock()
			cmd.(*redis.IntCmd).SetVal(1)
			return nil
		default:
			return next(ctx, cmd)
		}
	}
}

func (h *promotionHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		return next(ctx, cmds)
	}
}

func (h *promotionHook) state() (delayed, queued string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.delayed, h.queued
}

func TestPromoteDueStopsWhenContextCanceled(t *testing.T) {
	job, err := json.Marshal(domain.SendJob{RecipientID: "recipient-1", CampaignID: "campaign-1"})
	if err != nil {
		t.Fatal(err)
	}
	hook := &promotionHook{
		started: make(chan struct{}),
		release: make(chan struct{}),
		delayed: string(job),
	}
	rdb := redis.NewClient(&redis.Options{Addr: "unused.invalid:6379"})
	rdb.AddHook(hook)
	t.Cleanup(func() { _ = rdb.Close() })
	q := pipeline.NewQueue(rdb)

	ctx, cancel := context.WithCancel(context.Background())
	type outcome struct {
		n   int
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		n, err := q.PromoteDue(ctx, time.Now())
		done <- outcome{n: n, err: err}
	}()

	select {
	case <-hook.started:
	case <-time.After(time.Second):
		t.Fatal("promotion did not start")
	}
	cancel()

	var got outcome
	select {
	case got = <-done:
		close(hook.release)
	case <-time.After(25 * time.Millisecond):
		close(hook.release)
		select {
		case got = <-done:
		case <-time.After(time.Second):
			t.Fatal("promotion did not return")
		}
	}

	delayed, queued := hook.state()
	if !errors.Is(got.err, context.Canceled) || got.n != 0 || delayed == "" || queued != "" {
		t.Fatalf("after cancellation: promoted=%d err=%v delayed=%t queued=%t; want promoted=0 err=context canceled delayed=true queued=false",
			got.n, got.err, delayed != "", queued != "")
	}
}
