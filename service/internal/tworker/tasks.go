package tworker

import (
	"context"
	"errors"
	"log/slog"

	appErrors "github.com/m11ano/neurochar-experiments-3/service/internal/app/errors"
	taskEntity "github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/entity"
)

type CreateTaskPayload struct {
	Task *taskEntity.Task
}

type CreateTaskResponse struct{}

func (d *WorkerController) CreateTask(ctx context.Context, payload CreateTaskPayload) (*CreateTaskResponse, error) {
	err := d.taskFacade.Task.Create(ctx, payload.Task)
	if err != nil && !errors.Is(appErrors.ErrConflict, err) {
		return nil, err
	}

	return &CreateTaskResponse{}, nil
}

type UpdateTaskPayload struct {
	Task *taskEntity.Task
}

type UpdateTaskResponse struct{}

func (d *WorkerController) UpdateTask(ctx context.Context, payload UpdateTaskPayload) (*UpdateTaskResponse, error) {
	d.logger.Info("task for update", slog.Any("payload", payload))

	err := d.taskFacade.Task.Update(ctx, payload.Task)
	if err != nil {
		d.logger.Error("error in record updating", slog.Any("err", err))
	}

	return &UpdateTaskResponse{}, nil
}
