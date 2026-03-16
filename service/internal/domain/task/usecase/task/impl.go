package task

import (
	"context"
	"log/slog"

	"github.com/m11ano/neurochar-experiments-3/service/internal/app/config"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/usecase"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/usecase/task/workerpool"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/db"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/storage"
	temporalClient "github.com/m11ano/neurochar-experiments-3/service/internal/infra/temporal/client"
)

type UsecaseImpl struct {
	pkg            string
	logger         *slog.Logger
	cfg            config.Config
	s3Client       storage.Client
	dbMasterClient db.MasterClient
	repo           usecase.TaskRepository
	workerpool     *workerpool.Pool
	temporalClient temporalClient.Client
}

func NewUsecaseImpl(
	logger *slog.Logger,
	cfg config.Config,
	s3Client storage.Client,
	dbMasterClient db.MasterClient,
	repo usecase.TaskRepository,
	temporalClient temporalClient.Client,
) *UsecaseImpl {
	uc := &UsecaseImpl{
		pkg:            "Task.Usecase.Task",
		logger:         logger,
		cfg:            cfg,
		s3Client:       s3Client,
		dbMasterClient: dbMasterClient,
		repo:           repo,
		temporalClient: temporalClient,
	}

	return uc
}

func (uc *UsecaseImpl) Start(_ context.Context) error {
	var err error
	uc.workerpool, err = workerpool.New(context.Background(), workerpool.Config{Workers: 4, RecoverPanics: false})
	if err != nil {
		return err
	}

	uc.workerpool.Start()

	return nil
}

func (uc *UsecaseImpl) Stop(_ context.Context) error {
	uc.workerpool.Stop()

	return nil
}
