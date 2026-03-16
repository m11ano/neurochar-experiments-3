package task

import (
	"context"

	appErrors "github.com/m11ano/neurochar-experiments-3/service/internal/app/errors"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/entity"
)

func (uc *UsecaseImpl) Create(
	ctx context.Context,
	item *entity.Task,
) error {
	const op = "Create"

	err := uc.repo.Create(ctx, item)
	if err != nil {
		return appErrors.Chainf(err, "%s.%s", uc.pkg, op)
	}

	return nil
}

func (uc *UsecaseImpl) Update(
	ctx context.Context,
	item *entity.Task,
) error {
	const op = "Update"

	err := uc.repo.Update(ctx, item)
	if err != nil {
		return appErrors.Chainf(err, "%s.%s", uc.pkg, op)
	}

	return nil
}
