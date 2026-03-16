package pgclient

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type (
	txKey      struct{}
	txDepthKey struct{}
)

type clientImpl struct {
	serverID        string
	pool            Pool
	readOnly        bool
	defaultIsoLevel TxIsoLevel
	txKey           txKey
	txDepthKey      txDepthKey
	logger          *slog.Logger
}

// NewClientOpts - options for constructing pg client
type NewClientOpts struct {
	ReadOnly        bool
	DefaultIsoLevel TxIsoLevel
	Logger          *slog.Logger
	LogQueries      bool
}

// NewClient - create new pg client
func NewClient(ctx context.Context, serverID string, dsn string, opts NewClientOpts) (*clientImpl, error) {
	pgxCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse dsn: %w", err)
	}

	if opts.ReadOnly {
		pgxCfg.ConnConfig.RuntimeParams["default_transaction_read_only"] = "true"
	}

	if opts.LogQueries {
		pgxCfg.ConnConfig.Tracer = &pgxTracer{logger: opts.Logger}
	}

	dbpool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create pg connection pool: %w", err)
	}

	client := &clientImpl{
		serverID:        serverID,
		pool:            dbpool,
		readOnly:        opts.ReadOnly,
		defaultIsoLevel: ReadCommitted,
	}

	if opts.DefaultIsoLevel != "" {
		client.defaultIsoLevel = opts.DefaultIsoLevel
	}

	if opts.Logger != nil {
		client.logger = opts.Logger
	}

	return client, nil
}

var _ Client = (*clientImpl)(nil)

// ServerID - get server id
func (c *clientImpl) ServerID() string {
	return c.serverID
}

// Pool - get pool connection
func (c *clientImpl) Pool() Pool {
	return c.pool
}

// GetConn - get pg connection: pool or tx
func (c *clientImpl) GetConn(ctx context.Context) Conn {
	tx, ok := ctx.Value(c.txKey).(pgx.Tx)
	if ok && tx != nil {
		return tx
	}

	return c.pool
}

// Do - execute tx with default iso level
func (c *clientImpl) Do(ctx context.Context, fn func(context.Context) error) error {
	return c.DoWithIsoLvl(ctx, c.defaultIsoLevel, fn)
}

// DoWithIsoLvl - execute tx with iso level
func (c *clientImpl) DoWithIsoLvl(ctx context.Context, isoLvl TxIsoLevel, fn func(context.Context) error) error {
	tx, ok := ctx.Value(c.txKey).(pgx.Tx)
	if ok && tx != nil {
		depth, ok := ctx.Value(c.txKey).(int)
		if !ok {
			depth = 0
		}

		spName := fmt.Sprintf("sp_%d", depth+1)

		if _, err := tx.Exec(ctx, "SAVEPOINT "+spName); err != nil {
			return err
		}

		ctx = context.WithValue(ctx, c.txDepthKey, depth+1)

		err := fn(ctx)
		if err != nil {
			_, rbErr := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+spName)
			if rbErr != nil {
				if c.logger != nil {
					c.logger.ErrorContext(ctx, "rollback to savepoint failed", slog.Any("error", rbErr.Error()))
				}
			}

			return err
		}

		if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT "+spName); err != nil {
			return err
		}

		return nil
	}

	tx, err := c.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.TxIsoLevel(isoLvl)})
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, c.txKey, tx)
	ctx = context.WithValue(ctx, c.txDepthKey, 0)

	defer func() {
		if p := recover(); p != nil {
			rbErr := tx.Rollback(ctx)
			if rbErr != nil {
				if c.logger != nil {
					c.logger.ErrorContext(ctx, "rollback on panic failed", slog.Any("error", rbErr.Error()))
				}
			}

			panic(p)
		}
	}()

	err = fn(ctx)
	if err != nil {
		rbErr := tx.Rollback(ctx)
		if rbErr != nil {
			if c.logger != nil {
				c.logger.ErrorContext(ctx, "rollback failed", slog.Any("error", rbErr.Error()))
			}
		}

		return err
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return commitErr
	}

	return nil
}

// Close - close pool
func (c *clientImpl) Close() {
	c.pool.Close()
}
