package task

import (
	"context"

	"github.com/google/uuid"
	"github.com/m11ano/neurochar-experiments-3/service/internal/common/uctypes"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/entity"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/usecase"

	appErrors "github.com/m11ano/neurochar-experiments-3/service/internal/app/errors"
)

func (uc *UsecaseImpl) FindOneByID(
	ctx context.Context,
	id uuid.UUID,
	queryParams *uctypes.QueryGetOneParams,
) (*usecase.TaskDTO, error) {
	const op = "FindOneByID"

	item, err := uc.repo.FindOneByID(ctx, id, queryParams)
	if err != nil {
		return nil, appErrors.Chainf(err, "%s.%s", uc.pkg, op)
	}

	dto, err := uc.entitiesToDTO(ctx, []*entity.Task{item})
	if err != nil {
		return nil, appErrors.Chainf(err, "%s.%s", uc.pkg, op)
	}

	if len(dto) == 0 {
		return nil, appErrors.Chainf(appErrors.ErrInternal, "%s.%s", uc.pkg, op)
	}

	return dto[0], nil
}

func (uc *UsecaseImpl) FindList(
	ctx context.Context,
	listOptions *usecase.TaskListOptions,
	queryParams *uctypes.QueryGetListParams,
) ([]*usecase.TaskDTO, error) {
	const op = "FindList"

	items, err := uc.repo.FindList(ctx, listOptions, queryParams)
	if err != nil {
		return nil, appErrors.Chainf(err, "%s.%s", uc.pkg, op)
	}

	out, err := uc.entitiesToDTO(ctx, items)
	if err != nil {
		return nil, appErrors.Chainf(err, "%s.%s", uc.pkg, op)
	}

	return out, nil
}

func (uc *UsecaseImpl) QueueHasTasks(ctx context.Context, minSize uint64) (bool, error) {
	const op = "QueueHasTasks"

	value, err := uc.repo.CountTasksWithProcessStatus(ctx, false)
	if err != nil {
		return false, appErrors.Chainf(err, "%s.%s", uc.pkg, op)
	}

	return value > minSize, nil
}
