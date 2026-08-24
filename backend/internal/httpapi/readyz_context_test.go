package httpapi_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lumen/relay/internal/config"
	"github.com/lumen/relay/internal/httpapi"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var errPingContinued = errors.New("database ping continued after request cancellation")

type cancellationDriver struct{}

func (cancellationDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("open is not supported")
}

type pingProbe struct {
	started chan struct{}
	proceed chan struct{}
}

type cancellationConnector struct{ probe *pingProbe }

func (c cancellationConnector) Connect(context.Context) (driver.Conn, error) {
	return cancellationConn{probe: c.probe}, nil
}

func (cancellationConnector) Driver() driver.Driver { return cancellationDriver{} }

type cancellationConn struct{ probe *pingProbe }

func (cancellationConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (cancellationConn) Close() error { return nil }

func (cancellationConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (c cancellationConn) Ping(ctx context.Context) error {
	close(c.probe.started)
	<-c.probe.proceed
	if err := ctx.Err(); err != nil {
		return err
	}
	return errPingContinued
}

func TestReadyzHonorsRequestCancellation(t *testing.T) {
	probe := &pingProbe{started: make(chan struct{}), proceed: make(chan struct{})}
	sqlDB := sql.OpenDB(cancellationConnector{probe: probe})
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	engine := httpapi.New(httpapi.Deps{
		Cfg: config.Config{Env: "prod", CORSOrigins: []string{"http://example.test"}},
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:  db,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		engine.ServeHTTP(rec, req)
		close(done)
	}()
	select {
	case <-probe.started:
	case <-time.After(time.Second):
		t.Fatal("database ping did not start")
	}
	cancel()
	close(probe.proceed)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("readiness handler did not stop after request cancellation")
	}

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
	var body struct {
		OK bool   `json:"ok"`
		DB string `json:"db"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, rec.Body.String())
	}
	if body.OK {
		t.Fatalf("ok = true, want false; body=%s", rec.Body.String())
	}
	if body.DB != context.Canceled.Error() {
		t.Fatalf("db error = %q, want %q", body.DB, context.Canceled)
	}
}
