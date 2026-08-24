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
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", 884201); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer func() { _, err = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", 884201) }()

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
