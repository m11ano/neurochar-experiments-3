package tworker

import (
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	taskEntity "github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/entity"
	taskUC "github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/usecase"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/storage"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"google.golang.org/grpc"

	ocrpb "github.com/m11ano/neurochar-experiments-3/service/pkg/ocrpb"
)

type mlWorkerConn struct {
	mu          sync.RWMutex
	conn        *grpc.ClientConn
	ocr         ocrpb.OcrServiceClient
	connGen     uint64
	reconnectMu sync.Mutex
}

type WorkerController struct {
	instanceIndex int
	mlWorkerCfg   MlWorkersConfig
	taskFacade    *taskUC.Facade
	s3Client      storage.Client
	logger        *slog.Logger

	primaryConn  mlWorkerConn
	fallbackConn mlWorkerConn
}

func NewWorkerController(
	instanceIndex int,
	mlWorkerCfg MlWorkersConfig,
	taskFacade *taskUC.Facade,
	s3Client storage.Client,
	logger *slog.Logger,
) *WorkerController {
	return &WorkerController{
		instanceIndex: instanceIndex,
		mlWorkerCfg:   mlWorkerCfg,
		taskFacade:    taskFacade,
		s3Client:      s3Client,
		logger:        logger,
	}
}

const (
	TaskQueue    = "jobs"
	TaskQueueOcr = "jobs-ocr"
)

type JobPayload struct {
	TaskID     string
	Filename   string
	FileBucket string
	FileKey    string
}

type JobResult struct {
	Instance int
	Status   string
	Text     string
}

func (d *WorkerController) JobWorkflow(ctx workflow.Context, payload JobPayload) (*JobResult, error) {
	easyActivityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    0,
		},
	})

	ocrActivityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    15 * time.Second,
			MaximumAttempts:    0,
		},
		TaskQueue: TaskQueueOcr,
	})

	taskID, err := uuid.Parse(payload.TaskID)
	if err != nil {
		return nil, err
	}

	task, err := taskEntity.NewTask(payload.Filename, "temporal")
	if err != nil {
		return nil, err
	}

	task.ID = taskID

	var createTaskResult *CreateTaskResponse
	err = workflow.ExecuteActivity(easyActivityCtx, "CreateTask", CreateTaskPayload{Task: task}).Get(ctx, &createTaskResult)
	if err != nil {
		return nil, err
	}

	var readResult *ReadTextFromPDFResult
	err = workflow.ExecuteActivity(ocrActivityCtx, "ReadTextFromPDF", payload).Get(ctx, &readResult)
	if err != nil {
		return nil, err
	}
	task.IsProcessed = true
	task.Result = readResult.Text
	task.ProcessDurationMs = uint64(readResult.ProcessDuration.Milliseconds())
	if readResult.UsedFallback {
		task.Processor = "cpu_fallback_worker"
	} else {
		task.Processor = "gpu_worker"
	}

	var updateTaskResult *UpdateTaskResponse
	err = workflow.ExecuteActivity(easyActivityCtx, "UpdateTask", UpdateTaskPayload{Task: task}).Get(ctx, &updateTaskResult)
	if err != nil {
		return nil, err
	}

	return &JobResult{
		Instance: d.instanceIndex,
		Status:   "OK",
		Text:     readResult.Text,
	}, nil
}
