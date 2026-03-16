// Package pgclient contains pg client
package pgclient

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxIsoLevel string

const (
	Serializable    TxIsoLevel = "serializable"
	RepeatableRead  TxIsoLevel = "repeatable read"
	ReadCommitted   TxIsoLevel = "read committed"
	ReadUncommitted TxIsoLevel = "read uncommitted"
)

// Client - interface for pg client
type Client interface {
	ServerID() string
	Pool() Pool
	GetConn(ctx context.Context) Conn
	Do(ctx context.Context, fn func(context.Context) error) error
	DoWithIsoLvl(ctx context.Context, isoLvl TxIsoLevel, fn func(context.Context) error) error
	Close()
}

// Conn - interface for pg connection
type Conn interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Pool - interface for pg pool connection
type Pool interface {
	Acquire(ctx context.Context) (*pgxpool.Conn, error)
	AcquireAllIdle(ctx context.Context) []*pgxpool.Conn
	AcquireFunc(ctx context.Context, f func(*pgxpool.Conn) error) error
	Close()
	Config() *pgxpool.Config
	Stat() *pgxpool.Stat
	Reset()
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	CopyFrom(ctx context.Context, table pgx.Identifier, columns []string, rowSrc pgx.CopyFromSource) (int64, error)
	Ping(ctx context.Context) error
}
