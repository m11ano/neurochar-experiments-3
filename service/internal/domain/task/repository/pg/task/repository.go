package task

import (
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/db"
)

type Repository struct {
	pkg      string
	logger   *slog.Logger
	pgClient db.MasterClient
	qb       squirrel.StatementBuilderType
}

func NewRepository(logger *slog.Logger, pgClient db.MasterClient) *Repository {
	return &Repository{
		pkg:      "Task.repository.Task",
		logger:   logger,
		pgClient: pgClient,
		qb:       squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}
