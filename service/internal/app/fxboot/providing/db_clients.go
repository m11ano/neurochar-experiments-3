package providing

import (
	"context"
	"log/slog"

	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/db"
	"github.com/m11ano/neurochar-experiments-3/service/pkg/pgclient"
	"go.uber.org/fx"
)

// NewDBClients - fx module for db clients
func NewDBClients(masterDSN string, logQueries bool, logger *slog.Logger, shutdown fx.Shutdowner) db.MasterClient {
	master, err := pgclient.NewClient(
		context.Background(),
		"master",
		masterDSN,
		pgclient.NewClientOpts{
			Logger:     logger,
			LogQueries: logQueries,
		},
	)
	if err != nil {
		logger.Error("failed to create master client", slog.Any("error", err))
		// nolint
		_ = shutdown.Shutdown()
	}

	return master
}
