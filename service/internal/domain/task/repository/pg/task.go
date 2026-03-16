package pg

import (
	"time"

	"github.com/google/uuid"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/entity"
	"github.com/m11ano/neurochar-experiments-3/service/pkg/dbhelper"
)

const (
	TaskTable = "task"
)

var TaskTableFields = []string{}

func init() {
	TaskTableFields = dbhelper.ExtractDBFields(&TaskDBModel{})
}

type TaskDBModel struct {
	ID                uuid.UUID `db:"id"`
	Filename          string    `db:"file_name"`
	Method            string    `db:"method"`
	IsProcessed       bool      `db:"is_processed"`
	Result            string    `db:"result"`
	Processor         string    `db:"processor"`
	ProcessDurationMs uint64    `db:"process_duration_ms"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (db *TaskDBModel) ToEntity() *entity.Task {
	return &entity.Task{
		ID:                db.ID,
		Filename:          db.Filename,
		Method:            db.Method,
		IsProcessed:       db.IsProcessed,
		Result:            db.Result,
		Processor:         db.Processor,
		ProcessDurationMs: db.ProcessDurationMs,

		CreatedAt: db.CreatedAt,
		UpdatedAt: db.UpdatedAt,
	}
}

func MapTaskEntityToDBModel(entity *entity.Task) *TaskDBModel {
	return &TaskDBModel{
		ID:                entity.ID,
		Filename:          entity.Filename,
		Method:            entity.Method,
		IsProcessed:       entity.IsProcessed,
		Result:            entity.Result,
		Processor:         entity.Processor,
		ProcessDurationMs: entity.ProcessDurationMs,

		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}
