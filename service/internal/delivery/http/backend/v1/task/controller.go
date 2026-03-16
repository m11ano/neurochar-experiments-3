package task

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app/config"
	v1 "github.com/m11ano/neurochar-experiments-3/service/internal/delivery/http/backend/v1"
	"github.com/m11ano/neurochar-experiments-3/service/pkg/validation"

	taskUC "github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/usecase"
)

type Controller struct {
	pkg        string
	vldtr      *validator.Validate
	cfg        config.Config
	taskFacade *taskUC.Facade
}

func NewController(
	cfg config.Config,
	taskFacade *taskUC.Facade,
) *Controller {
	controller := &Controller{
		pkg:        "httpController.Task",
		vldtr:      validation.New(),
		cfg:        cfg,
		taskFacade: taskFacade,
	}
	return controller
}

func RegisterRoutes(groups *v1.Groups, ctrl *Controller) {
	const url = "tasks"

	routeGroup := groups.Default.Group(fmt.Sprintf("/%s", url))

	routeGroup.Get("/queue-metrics", ctrl.QueueMetricsHandler)

	routeGroup.Post("/temporal-coordination", ctrl.TemporalCoordinationHandler)
}
