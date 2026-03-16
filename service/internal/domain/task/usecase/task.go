package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/m11ano/neurochar-experiments-3/service/internal/common/uctypes"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/entity"
)

type TaskListOptions struct{}

type TaskDTO struct {
	Task *entity.Task
}

type TaskUsecase interface {
	Start(ctx context.Context) (resErr error)

	Stop(ctx context.Context) (resErr error)

	FindOneByID(
		ctx context.Context,
		id uuid.UUID,
		queryParams *uctypes.QueryGetOneParams,
	) (resTask *TaskDTO, resErr error)

	FindList(
		ctx context.Context,
		listOptions *TaskListOptions,
		queryParams *uctypes.QueryGetListParams,
	) (resItems []*TaskDTO, resErr error)

	StartTaskWithBaselineCoordination(
		ctx context.Context,
		filename string,
		filedata []byte,
		count int,
	) (resErr error)

	StartTaskWithTemporalCoordination(
		ctx context.Context,
		filename string,
		filedata []byte,
		count int,
	) (resErr error)

	Create(ctx context.Context, item *entity.Task) (resErr error)

	Update(ctx context.Context, item *entity.Task) (resErr error)

	QueueHasTasks(ctx context.Context, minSize uint64) (res bool, resErr error)
}

type TaskRepository interface {
	FindOneByID(
		ctx context.Context,
		id uuid.UUID,
		queryParams *uctypes.QueryGetOneParams,
	) (account *entity.Task, err error)

	FindList(
		ctx context.Context,
		listOptions *TaskListOptions,
		queryParams *uctypes.QueryGetListParams,
	) (items []*entity.Task, err error)

	FindPagedList(
		ctx context.Context,
		listOptions *TaskListOptions,
		queryParams *uctypes.QueryGetListParams,
	) (items []*entity.Task, total uint64, err error)

	Create(ctx context.Context, item *entity.Task) (err error)

	Update(ctx context.Context, item *entity.Task) (err error)

	CountTasksWithProcessStatus(ctx context.Context, isProcessed bool) (res uint64, resErr error)
}
