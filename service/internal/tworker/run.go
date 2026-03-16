package tworker

import (
	"context"
	"fmt"
	"log/slog"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	taskUC "github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/usecase"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/storage"
	temporalWorker "github.com/m11ano/neurochar-experiments-3/service/internal/infra/temporal/worker"
)

type MlWorkersConfig struct {
	Service         string
	Readiness       string
	FallbackService string
}

func RunWorker(
	ctx context.Context,
	workerClient temporalWorker.WorkerClient,
	mlWorkerCfg MlWorkersConfig,
	s3Client storage.Client,
	taskFacade *taskUC.Facade,
	logger *slog.Logger,
) error {
	errCh := make(chan error, 2)

	deps := NewWorkerController(0, mlWorkerCfg, taskFacade, s3Client, logger)

	defaultWorker := worker.New(workerClient, TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	ocrWorker := worker.New(workerClient, TaskQueueOcr, worker.Options{
		MaxConcurrentActivityExecutionSize: 2,
	})

	defaultWorker.RegisterWorkflowWithOptions(deps.JobWorkflow, workflow.RegisterOptions{
		Name: "JobWorkflow",
	})

	defaultWorker.RegisterActivityWithOptions(deps.CreateTask, activity.RegisterOptions{
		Name: "CreateTask",
	})

	defaultWorker.RegisterActivityWithOptions(deps.UpdateTask, activity.RegisterOptions{
		Name: "UpdateTask",
	})

	ocrWorker.RegisterActivityWithOptions(deps.ReadTextFromPDF, activity.RegisterOptions{
		Name: "ReadTextFromPDF",
	})

	go func() {
		if err := defaultWorker.Run(worker.InterruptCh()); err != nil {
			errCh <- fmt.Errorf("default worker stopped: %w", err)
		}
	}()

	go func() {
		if err := ocrWorker.Run(worker.InterruptCh()); err != nil {
			errCh <- fmt.Errorf("ocr worker stopped: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}
