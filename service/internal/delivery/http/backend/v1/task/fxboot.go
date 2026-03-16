package task

import (
	"go.uber.org/fx"
)

// FxModule - fx module
var FxModule = fx.Options(
	fx.Provide(NewController),
	fx.Invoke(RegisterRoutes),
)
