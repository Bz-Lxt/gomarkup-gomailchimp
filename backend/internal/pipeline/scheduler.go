package pipeline

import (
	"context"
	"log/slog"
	"time"

	"github.com/lumen/relay/internal/clock"
	"github.com/lumen/relay/internal/repo"
)

type Starter func(tenantID, campaignID string) error

type Scheduler struct {
	Store   repo.Store
	Queue   *Queue
	Log     *slog.Logger
	StartFn Starter
}

func (s *Scheduler) Run(ctx context.Context) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			now := clock.Now()
			if _, err := s.Queue.PromoteDue(ctx, now); err != nil && s.Log != nil {
				s.Log.Warn("promote", "err", err)
			}
			due, err := s.Store.DueScheduled(now)
			if err != nil {
				continue
			}
			for _, c := range due {
				if s.StartFn != nil {
					if err := s.StartFn(c.TenantID, c.ID); err != nil && s.Log != nil {
						s.Log.Warn("start scheduled", "id", c.ID, "err", err)
					}
				}
			}
		}
	}
}
