package db

import (
	"fmt"

	"sensorpanel/db/migrations"

	"github.com/pressly/goose/v3"
)

func Migrate(database *Database) error {
	if database == nil {
		return fmt.Errorf("database is not initialized")
	}

	sqlDB, err := database.SQLDB()
	if err != nil {
		return fmt.Errorf("resolve sql db handle: %w", err)
	}
	if sqlDB == nil {
		return fmt.Errorf("database is not initialized")
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("run goose migrations: %w", err)
	}

	return ping(sqlDB)
}

func ping(sqlDB interface{ Ping() error }) error {
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	return nil
}
