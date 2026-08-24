package migrate_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"

	"github.com/lumen/relay/internal/migrate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const failingDriverName = "lumen-migrate-catalog-failure"

var (
	registerFailingDriver sync.Once
	errCatalogUnavailable = errors.New("catalog unavailable")
)

func TestUpReturnsDriverInitializationError(t *testing.T) {
	registerFailingDriver.Do(func() {
		sql.Register(failingDriverName, catalogFailureDriver{})
	})

	sqlDB, err := sql.Open(failingDriverName, "")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(4)

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = migrate.Up(db, migrate.Files, migrate.Dir)
	if err == nil {
		t.Fatal("migration setup error was lost")
	}
	if !errors.Is(err, errCatalogUnavailable) {
		t.Fatalf("unexpected error: %v", err)
	}
}

type catalogFailureDriver struct{}

func (catalogFailureDriver) Open(string) (driver.Conn, error) {
	return &catalogFailureConn{}, nil
}

type catalogFailureConn struct{}

func (*catalogFailureConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (*catalogFailureConn) Close() error { return nil }

func (*catalogFailureConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (*catalogFailureConn) Ping(context.Context) error { return errCatalogUnavailable }

func (*catalogFailureConn) ExecContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return driver.RowsAffected(1), nil
}

func (*catalogFailureConn) QueryContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, errCatalogUnavailable
}
