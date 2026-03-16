package task

import (
	"context"
	"log/slog"

	"github.com/m11ano/neurochar-experiments-3/service/internal/app"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app/fxboot/invoking"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/usecase"
)

// Init - init domain
func Init(_ app.ID, logger *slog.Logger, taskUC usecase.TaskUsecase) invoking.InvokeInit {
	return invoking.InvokeInit{
		StartBeforeOpen: func(ctx context.Context) error {
			return taskUC.Start(ctx)
		},
		Stop: func(ctx context.Context) error {
			return taskUC.Stop(ctx)
		},
	}
}
