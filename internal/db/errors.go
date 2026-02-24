package db

import (
	"fmt"
	"log"
	"strings"
	"sync/atomic"
)

var sqliteLockErrors atomic.Uint64

func IsSQLiteLockedError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "sqlite_busy") ||
		strings.Contains(msg, "sql_busy")
}

func RecordSQLiteError(op string, err error) {
	if !IsSQLiteLockedError(err) {
		return
	}

	count := sqliteLockErrors.Add(1)
	log.Printf("sqlite lock contention op=%s count=%d err=%v", op, count, err)
}

func SQLiteLockErrorCount() uint64 {
	return sqliteLockErrors.Load()
}

func WrapWithOp(op string, err error) error {
	if err == nil {
		return nil
	}

	RecordSQLiteError(op, err)

	return fmt.Errorf("%s: %w", op, err)
}
