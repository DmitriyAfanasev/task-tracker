package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var files embed.FS

// Up применяет встроенные миграции при старте приложения.
func Up(db *sql.DB, log *slog.Logger) error {
	goose.SetBaseFS(files)
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("get migration version: %w", err)
	}
	log.Info("migrations applied", "version", version)
	return nil
}
