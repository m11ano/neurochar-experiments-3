package entity

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID                uuid.UUID
	Filename          string
	Method            string
	IsProcessed       bool
	Result            string
	Processor         string
	ProcessDurationMs uint64

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (item *Task) Version() int64 {
	return item.UpdatedAt.UnixMicro()
}

func NewTask(
	filename string,
	method string,
) (*Task, error) {
	timeNow := time.Now().Truncate(time.Microsecond)
	task := &Task{
		ID:        uuid.New(),
		Filename:  filename,
		Method:    method,
		CreatedAt: timeNow,
		UpdatedAt: timeNow,
	}

	return task, nil
}
