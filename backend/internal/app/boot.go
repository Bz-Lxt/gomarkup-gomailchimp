package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lumen/relay/internal/bounce"
	"github.com/lumen/relay/internal/config"
	"github.com/lumen/relay/internal/httpapi"
	"github.com/lumen/relay/internal/limiter"
	"github.com/lumen/relay/internal/logger"
	"github.com/lumen/relay/internal/migrate"
	"github.com/lumen/relay/internal/pipeline"
	"github.com/lumen/relay/internal/provider"
	"github.com/lumen/relay/internal/repo"
	"github.com/lumen/relay/internal/seed"
	"github.com/lumen/relay/internal/service"
	"github.com/lumen/relay/internal/tracker"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Runtime struct {
	Cfg   config.Config
	Log   *slog.Logger
	DB    *gorm.DB
	RDB   *redis.Client
	Store repo.Store
	Q     *pipeline.Queue
	Camp  service.Campaigns
	Send  provider.Sender
}

func Boot() (*Runtime, error) {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel, cfg.Env)
	db, err := repo.Open(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := migrate.Up(db, migrate.Files, migrate.Dir); err != nil {
		return nil, err
	}
	if err := seed.Run(db); err != nil {
		return nil, err
	}
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	store := repo.Store{DB: db}
	q := pipeline.NewQueue(rdb)
	rb := limiter.NewRedis(rdb)
	lim := &limiter.Multi{AllowFn: rb.Allow}
	send := pickSender(cfg)
	chs, _ := store.Channels(seed.TenantID)
	var hs []provider.Health
	for _, c := range chs {
		hs = append(hs, provider.Health{Key: c.Key, Weight: c.Weight, Score: c.Health, State: c.State})
	}
	bal := provider.NewBalancer(hs)
	camp := service.Campaigns{Store: store, Q: q, Cfg: cfg, Send: send, Bal: bal, Lim: lim}
	return &Runtime{Cfg: cfg, Log: log, DB: db, RDB: rdb, Store: store, Q: q, Camp: camp, Send: send}, nil
}

func pickSender(cfg config.Config) provider.Sender {
	switch cfg.MailProvider {
	case "mock":
		return provider.NewMock()
	case "ses":
		return provider.NewSES(cfg.AWSRegion, cfg.AWSAccessKey, cfg.AWSSecretKey)
	default:
		return provider.NewSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPStartTLS, "smtp")
	}
}

func (rt *Runtime) RunAPI() error {
	r := httpapi.New(httpapi.Deps{
		Cfg: rt.Cfg, Log: rt.Log, DB: rt.DB, RDB: rt.RDB, Store: rt.Store,
		Auth: service.Auth{Store: rt.Store, Cfg: rt.Cfg},
		Imp:  service.Importer{Store: rt.Store},
		Camp: rt.Camp, Q: rt.Q,
	})
	return listen(rt.Cfg.HTTPAddr, r, rt.Log)
}

func (rt *Runtime) RunTracker() error {
	g := tracker.Gateway{Cfg: rt.Cfg, Log: rt.Log, Camp: rt.Camp}
	return listen(rt.Cfg.HTTPAddr, g.Engine(), rt.Log)
}

func (rt *Runtime) RunSender() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	sched := &pipeline.Scheduler{
		Store: rt.Store, Queue: rt.Q, Log: rt.Log,
		StartFn: func(tenant, id string) error {
			_, err := rt.Camp.Transit(tenant, id, "running", "")
			return err
		},
	}
	go sched.Run(ctx)
	go rt.pollBounce(ctx)
	pool := &pipeline.Pool{Workers: rt.Cfg.SenderWorkers, Queue: rt.Q, Handle: rt.Camp.HandleJob, Log: rt.Log}
	rt.Log.Info("sender start", "workers", rt.Cfg.SenderWorkers, "provider", rt.Cfg.MailProvider)
	pool.Start(ctx)
	<-ctx.Done()
	rt.Log.Info("sender draining")
	done := make(chan struct{})
	go func() { pool.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(25 * time.Second):
		rt.Log.Warn("sender drain timeout")
	}
	return nil
}

func (rt *Runtime) pollBounce(ctx context.Context) {
	src := bounce.Select(rt.Cfg.BounceSource, &bounce.MockFeeder{}, bounce.IMAPSource{}, &bounce.WebhookSource{})
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			evs, err := src.Poll(ctx)
			if err != nil {
				rt.Log.Warn("bounce poll", "err", err)
				continue
			}
			for _, ev := range evs {
				rt.Log.Info("bounce", "email", ev.Email, "class", ev.Class, "src", ev.Source)
			}
		}
	}
}

func listen(addr string, h http.Handler, log *slog.Logger) error {
	srv := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 8 * time.Second}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		return err
	case <-sig:
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		log.Info("http shutdown", "addr", addr)
		return srv.Shutdown(ctx)
	}
}
