package contracts

import (
	"context"
	"database/sql"
)

type Tx interface {
	Commit() error
	Rollback() error
}

type Database interface {
	DriverName() string
	Begin(ctx context.Context) (context.Context, Tx, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
}

type Rows interface {
	Close() error
	Next() bool
	Scan(dest ...any) error
	Err() error
}
