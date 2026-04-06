// Package db contains logic to open a database connection.
package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const DefaultDatabaseURI = "~/.config/sensorpanel.db.sqlite3"

const defaultSQLitePragmaOpts = "?mode=rwc" +
	"&_pragma=journal_mode(WAL)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=busy_timeout(2000)" +
	"&_pragma=cache_size(-8192)" +
	"&_pragma=temp_store(MEMORY)" +
	"&_pragma=wal_autocheckpoint(1000)" +
	"&_pragma=journal_size_limit(67108864)" +
	"&_pragma=mmap_size(134217728)" +
	"&_pragma=foreign_keys(ON)"

type Config struct {
	DatabaseURI string
	Environment string
}

type Database struct {
	*gorm.DB
}

func ResolveSQLitePath(databaseURI string) (string, error) {
	return normalizeSQLitePath(databaseURI)
}

func New(cfg Config) (*Database, error) {
	path, err := normalizeSQLitePath(cfg.DatabaseURI)
	if err != nil {
		return nil, err
	}
	dsn := "file:" + path + defaultSQLitePragmaOpts

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:      newLogger(cfg.Environment),
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db handle: %w", err)
	}

	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxIdleTime(3 * time.Minute)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return &Database{DB: db}, nil
}

func (d *Database) SQLDB() (*sql.DB, error) {
	if d == nil || d.DB == nil {
		return nil, nil
	}

	sqlDB, err := d.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db handle: %w", err)
	}

	return sqlDB, nil
}

func (d *Database) Close() error {
	sqlDB, err := d.SQLDB()
	if err != nil {
		return err
	}

	if sqlDB == nil {
		return nil
	}

	return sqlDB.Close()
}

func normalizeSQLitePath(databaseURI string) (string, error) {
	path := strings.TrimSpace(databaseURI)
	if path == "" {
		path = DefaultDatabaseURI
	}

	expandedPath, err := expandUserPath(path)
	if err != nil {
		return "", err
	}
	path = expandedPath

	if strings.Contains(path, "://") {
		return "", fmt.Errorf("DATABASE_URI must be a sqlite file path, got: %s", path)
	}
	if strings.Contains(path, "?") {
		return "", fmt.Errorf("DATABASE_URI must not include query params, got: %s", path)
	}

	if err := ensureDirForPath(path); err != nil {
		return "", err
	}

	return path, nil
}

func expandUserPath(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir for DATABASE_URI: %w", err)
		}
		return home, nil
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir for DATABASE_URI: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}

func ensureDirForPath(dbPath string) error {
	clean := filepath.Clean(dbPath)
	dir := filepath.Dir(clean)
	if dir == "." || dir == "" {
		return nil
	}

	return os.MkdirAll(dir, 0o755)
}

func newLogger(environment string) logger.Interface {
	logLevel := logger.Warn
	hideQueryParams := true
	colorful := false

	if strings.EqualFold(strings.TrimSpace(environment), "development") {
		logLevel = logger.Info
		hideQueryParams = false
		colorful = true
	}

	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             150 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      hideQueryParams,
			Colorful:                  colorful,
		},
	)
}
