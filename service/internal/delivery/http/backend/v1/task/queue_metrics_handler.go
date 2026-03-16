package task

import (
	"github.com/gofiber/fiber/v2"
	appErrors "github.com/m11ano/neurochar-experiments-3/service/internal/app/errors"
)

type QueueMetricsHandlerResponse struct {
	Value uint64 `json:"value"`
}

func (ctrl *Controller) QueueMetricsHandler(c *fiber.Ctx) error {
	const op = "QueueMetricsHandler"

	hasTasks, err := ctrl.taskFacade.Task.QueueHasTasks(c.Context(), 3)
	if err != nil {
		return appErrors.Chainf(
			appErrors.ErrInternal.WithWrap(err),
			"%s.%s", ctrl.pkg, op,
		)
	}

	out := QueueMetricsHandlerResponse{}

	if hasTasks {
		out.Value = 1
	}

	return c.JSON(out)
}
