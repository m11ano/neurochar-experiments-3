package task

import (
	"io"

	"github.com/gofiber/fiber/v2"
	appErrors "github.com/m11ano/neurochar-experiments-3/service/internal/app/errors"
)

func (ctrl *Controller) TemporalCoordinationHandler(c *fiber.Ctx) error {
	const op = "TemporalCoordinationHandler"

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return appErrors.Chainf(
			appErrors.ErrBadRequest.WithWrap(err).WithHints("form field `file` is required"),
			"%s.%s", ctrl.pkg, op,
		)
	}

	if fileHeader.Header.Get("Content-Type") != "application/pdf" {
		return appErrors.Chainf(
			appErrors.ErrBadRequest.WithHints("only PDF files are allowed"),
			"%s.%s", ctrl.pkg, op,
		)
	}

	f, err := fileHeader.Open()
	if err != nil {
		return appErrors.Chainf(
			appErrors.ErrInternal.WithWrap(err),
			"%s.%s", ctrl.pkg, op,
		)
	}
	// nolint
	defer f.Close()

	fileData, err := io.ReadAll(f)
	if err != nil {
		return appErrors.Chainf(
			appErrors.ErrInternal.WithWrap(err),
			"%s.%s", ctrl.pkg, op,
		)
	}

	count := c.QueryInt("count", 1)

	err = ctrl.taskFacade.Task.StartTaskWithTemporalCoordination(c.Context(), fileHeader.Filename, fileData, count)
	if err != nil {
		return appErrors.Chainf(
			appErrors.ErrInternal.WithWrap(err),
			"%s.%s", ctrl.pkg, op,
		)
	}

	return nil
}
