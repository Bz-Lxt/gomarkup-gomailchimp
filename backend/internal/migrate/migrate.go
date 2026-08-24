package migrate

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

func Up(db *gorm.DB, files embed.FS, dir string) (err error) {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migrate conn: %w", err)
	}
	defer conn.Close()

	lock := func() error {
		_, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", 884201)
		return err
	}
	// unlock uses context.Background() so the timeout ctx cannot cancel it.
	unlock := func() error {
		_, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", 884201)
		return err
	}
	migrateFn := func() error {
		// golang-migrate needs *sql.DB; serialize via the held session lock above.
		driver, err := pg.WithInstance(sqlDB, &pg.Config{})
		if err != nil {
			return fmt.Errorf("migrate driver: %w", err)
		}
		src, err := iofs.New(files, dir)
		if err != nil {
			return fmt.Errorf("migrate fs: %w", err)
		}
		m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
		if err != nil {
			return fmt.Errorf("migrate new: %w", err)
		}
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return err
		}
		return nil
	}

	return runLocked(lock, unlock, migrateFn)
}

// runLocked acquires an advisory lock, runs migrateFn, then releases the lock.
//
// The first error encountered — from the lock, the migration, or the unlock —
// is the one returned. A successful unlock must NEVER overwrite a migration
// error; the deployment retry logic needs the original failure so that boot
// stops on the first migration error instead of proceeding to seed and
// surfacing a misleading "relation does not exist" downstream.
func runLocked(lock, unlock, migrateFn func() error) (err error) {
	if lockErr := lock(); lockErr != nil {
		return fmt.Errorf("advisory lock: %w", lockErr)
	}
	defer func() {
		if unlockErr := unlock(); unlockErr != nil && err == nil {
			err = unlockErr
		}
	}()
	return migrateFn()
}
