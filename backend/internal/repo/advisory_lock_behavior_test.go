package repo_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lumen/relay/internal/repo"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAdvisoryLockHeldUntilRelease(t *testing.T) {
	db, sqlDB := advisoryTestDB(t)
	defer sqlDB.Close()

	releaseFirst, err := repo.AdvisoryLock(db, 731)
	if err != nil {
		t.Fatalf("acquire first advisory lock: %v", err)
	}

	type acquisition struct {
		release func()
		err     error
	}
	second := make(chan acquisition, 1)
	go func() {
		release, err := repo.AdvisoryLock(db, 731)
		second <- acquisition{release: release, err: err}
	}()

	select {
	case got := <-second:
		if got.release != nil {
			got.release()
		}
		t.Fatalf("second holder entered before first release: %v", got.err)
	case <-time.After(75 * time.Millisecond):
	}

	releaseFirst()
	select {
	case got := <-second:
		if got.err != nil {
			t.Fatalf("acquire after release: %v", got.err)
		}
		if got.release == nil {
			t.Fatal("acquire after release returned no release function")
		}
		got.release()
	case <-time.After(time.Second):
		t.Fatal("second holder stayed blocked after first release")
	}
}

func advisoryTestDB(t *testing.T) (*gorm.DB, *sql.DB) {
	t.Helper()
	connector := &advisoryConnector{state: newAdvisoryState()}
	sqlDB := sql.OpenDB(connector)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		sqlDB.Close()
		t.Fatalf("open gorm db: %v", err)
	}
	return db, sqlDB
}

type advisoryConnector struct {
	state *advisoryState
	next  atomic.Int64
}

func (c *advisoryConnector) Connect(context.Context) (driver.Conn, error) {
	return &advisoryConn{id: c.next.Add(1), state: c.state}, nil
}

func (c *advisoryConnector) Driver() driver.Driver { return advisoryDriver{} }

type advisoryDriver struct{}

func (advisoryDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use connector")
}

type advisoryState struct {
	mu      sync.Mutex
	owner   int64
	depth   int
	changed chan struct{}
}

func newAdvisoryState() *advisoryState {
	return &advisoryState{changed: make(chan struct{})}
}

type advisoryConn struct {
	id    int64
	state *advisoryState
}

func (c *advisoryConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}

func (c *advisoryConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions unsupported")
}

func (c *advisoryConn) Close() error {
	c.state.releaseAll(c.id)
	return nil
}

func (c *advisoryConn) ExecContext(ctx context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	switch {
	case strings.Contains(query, "pg_advisory_lock"):
		return driver.RowsAffected(1), c.state.acquire(ctx, c.id)
	case strings.Contains(query, "pg_advisory_unlock"):
		c.state.release(c.id)
		return driver.RowsAffected(1), nil
	default:
		return nil, errors.New("unexpected query")
	}
}

func (s *advisoryState) acquire(ctx context.Context, id int64) error {
	for {
		s.mu.Lock()
		if s.owner == 0 || s.owner == id {
			s.owner = id
			s.depth++
			s.mu.Unlock()
			return nil
		}
		changed := s.changed
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}

func (s *advisoryState) release(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner != id || s.depth == 0 {
		return
	}
	s.depth--
	if s.depth == 0 {
		s.owner = 0
		close(s.changed)
		s.changed = make(chan struct{})
	}
}

func (s *advisoryState) releaseAll(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner != id {
		return
	}
	s.owner = 0
	s.depth = 0
	close(s.changed)
	s.changed = make(chan struct{})
}
