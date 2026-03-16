package backend

import (
	v1 "github.com/m11ano/neurochar-experiments-3/service/internal/delivery/http/backend/v1"
	"github.com/m11ano/neurochar-experiments-3/service/internal/delivery/http/backend/v1/task"
	"github.com/m11ano/neurochar-experiments-3/service/pkg/validation"
	"go.uber.org/fx"
)

// FxModule - fx module
var FxModule = fx.Options(
	fx.Provide(validation.New),
	fx.Options(
		fx.Provide(v1.ProvideGroups),
		task.FxModule,
	),
)
