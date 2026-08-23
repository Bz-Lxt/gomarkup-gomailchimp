package pipeline

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/lumen/relay/internal/domain"
)

type Handler func(ctx context.Context, job domain.SendJob) error

type Pool struct {
	Workers int
	Queue   *Queue
	Handle  Handler
	Log     *slog.Logger
	wg      sync.WaitGroup
}

func (p *Pool) Start(ctx context.Context) {
	n := p.Workers
	if n <= 0 {
		n = 8
	}
	for i := 0; i < n; i++ {
		p.wg.Add(1)
		go p.loop(ctx, i)
	}
}

func (p *Pool) Wait() { p.wg.Wait() }

func (p *Pool) loop(ctx context.Context, id int) {
	defer p.wg.Done()
	for {
		if ctx.Err() != nil {
			return
		}
		job, ok, err := p.Queue.Pop(ctx, 2*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			p.Log.Warn("queue pop", "worker", id, "err", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if !ok {
			continue
		}
		if err := p.Handle(ctx, job); err != nil && p.Log != nil {
			p.Log.Warn("job", "worker", id, "recipient", job.RecipientID, "err", err)
		}
	}
}
