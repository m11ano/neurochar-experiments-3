package task

import (
	"go.uber.org/fx"

	taskRepo "github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/repository/pg/task"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/usecase"
	taskUC "github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/usecase/task"
)

// FxModule - fx module
var FxModule = fx.Module(
	"task_module",

	// repositories
	fx.Provide(
		fx.Private,
		fx.Annotate(taskRepo.NewRepository, fx.As(new(usecase.TaskRepository))),
	),

	// usecases
	fx.Provide(
		fx.Annotate(taskUC.NewUsecaseImpl, fx.As(new(usecase.TaskUsecase))),
	),

	// facade
	fx.Provide(
		usecase.NewFacade,
	),

	// init
	fx.Provide(
		fx.Annotate(Init, fx.ResultTags(`group:"InvokeInit"`)),
	),
)
