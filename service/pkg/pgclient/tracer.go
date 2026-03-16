package pgclient

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

type pgxTracer struct {
	logger *slog.Logger
}

func (t *pgxTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	t.logger.InfoContext(
		ctx, "start of query",
		slog.String("sql.query", data.SQL),
		slog.Any("sql.args", data.Args),
	)

	return ctx
}

func (t *pgxTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	t.logger.InfoContext(
		ctx,
		"end of query",
		slog.String("sql.query.ctag", fmt.Sprintf("%v", data.CommandTag)),
		slog.Any("sql.query.error", data.Err),
	)
}
