package task

import (
	"context"

	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/entity"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/usecase"
)

func (uc *UsecaseImpl) entitiesToDTO(
	ctx context.Context,
	items []*entity.Task,
) ([]*usecase.TaskDTO, error) {
	const op = "entitiesToDTO"

	out := make([]*usecase.TaskDTO, 0, len(items))

	for _, item := range items {
		resItem := &usecase.TaskDTO{
			Task: item,
		}

		out = append(out, resItem)
	}

	return out, nil
}
