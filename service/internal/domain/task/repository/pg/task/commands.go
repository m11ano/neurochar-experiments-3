package task

import (
	"context"
	"log/slog"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	appErrors "github.com/m11ano/neurochar-experiments-3/service/internal/app/errors"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/entity"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/repository/pg"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/loghandler"
	"github.com/m11ano/neurochar-experiments-3/service/pkg/dbhelper"
)

func (r *Repository) Create(ctx context.Context, item *entity.Task) error {
	const op = "Create"

	dataMap, err := dbhelper.DBModelToMap(pg.MapTaskEntityToDBModel(item))
	if err != nil {
		r.logger.ErrorContext(loghandler.WithSource(ctx), "convert struct to db map", slog.Any("error", err))
		return appErrors.Chainf(appErrors.ErrInternal.WithWrap(err), "%s.%s", r.pkg, op)
	}

	query, args, err := r.qb.Insert(pg.TaskTable).SetMap(dataMap).ToSql()
	if err != nil {
		r.logger.ErrorContext(loghandler.WithSource(ctx), "building query", slog.Any("error", err))
		return appErrors.Chainf(appErrors.ErrInternal.WithWrap(err), "%s.%s", r.pkg, op)
	}

	_, err = r.pgClient.GetConn(ctx).Exec(ctx, query, args...)
	if err != nil {
		convErr, ok := appErrors.ConvertPgxToAppErr(err)
		if !ok {
			r.logger.ErrorContext(loghandler.WithSource(ctx), "executing query", slog.Any("error", err))
		}
		return appErrors.Chainf(convErr, "%s.%s", r.pkg, op)
	}

	return nil
}

func (r *Repository) Update(ctx context.Context, item *entity.Task) error {
	const op = "Update"

	// currentUpdatedAt := item.UpdatedAt
	timeNow := time.Now().Truncate(time.Microsecond)

	dataMap, err := dbhelper.DBModelToMap(pg.MapTaskEntityToDBModel(item))
	if err != nil {
		r.logger.ErrorContext(loghandler.WithSource(ctx), "convert struct to db map", slog.Any("error", err))
		return appErrors.Chainf(appErrors.ErrInternal.WithWrap(err), "%s.%s", r.pkg, op)
	}
	delete(dataMap, "id")
	dataMap["updated_at"] = timeNow

	err = r.pgClient.Do(ctx, func(ctx context.Context) error {
		checkQuery, checkArgs, err := r.qb.Select("id").From(pg.TaskTable).Where(squirrel.Eq{"id": item.ID}).ToSql()
		if err != nil {
			r.logger.ErrorContext(loghandler.WithSource(ctx), "building check query", slog.Any("error", err))
			return appErrors.Chainf(appErrors.ErrInternal.WithWrap(err), "%s.%s", r.pkg, op)
		}

		var checkID uuid.UUID
		err = r.pgClient.GetConn(ctx).QueryRow(ctx, checkQuery, checkArgs...).Scan(&checkID)
		if err != nil {
			convErr, ok := appErrors.ConvertPgxToAppErr(err)
			if !ok {
				r.logger.ErrorContext(loghandler.WithSource(ctx), "executing check query", slog.Any("error", err))
			}
			return appErrors.Chainf(convErr, "%s.%s", r.pkg, op)
		}

		// updWhere := squirrel.And{squirrel.Eq{"id": item.ID}}
		// if !currentUpdatedAt.IsZero() {
		// 	updWhere = append(updWhere, squirrel.Eq{"updated_at": currentUpdatedAt})
		// }

		updQuery, updArgs, err := r.qb.Update(pg.TaskTable).Where(squirrel.Eq{"id": item.ID}).SetMap(dataMap).ToSql()
		if err != nil {
			r.logger.ErrorContext(loghandler.WithSource(ctx), "building query", slog.Any("error", err))
			return appErrors.Chainf(appErrors.ErrInternal.WithWrap(err), "%s.%s", r.pkg, op)
		}

		cmdTag, err := r.pgClient.GetConn(ctx).Exec(ctx, updQuery, updArgs...)
		if err != nil {
			convErr, ok := appErrors.ConvertPgxToAppErr(err)
			if !ok {
				r.logger.ErrorContext(loghandler.WithSource(ctx), "executing query", slog.Any("error", err))
			}
			return appErrors.Chainf(convErr, "%s.%s", r.pkg, op)
		}

		if cmdTag.RowsAffected() == 0 {
			return appErrors.Chainf(appErrors.ErrConflict, "%s.%s", r.pkg, op)
		}

		item.UpdatedAt = timeNow

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
