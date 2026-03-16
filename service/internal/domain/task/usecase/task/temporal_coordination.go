package task

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	appErrors "github.com/m11ano/neurochar-experiments-3/service/internal/app/errors"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/storage"
	"github.com/m11ano/neurochar-experiments-3/service/internal/tworker"
	"go.temporal.io/sdk/client"
)

func (uc *UsecaseImpl) StartTaskWithTemporalCoordination(
	ctx context.Context,
	filename string,
	filedata []byte,
	count int,
) error {
	const op = "StartTaskWithTemporalCoordination"

	fileKey, _, _, err := uc.s3Client.UploadFileByBytes(ctx, storage.BucketCommonFiles, filename, filedata, nil)
	if err != nil {
		return appErrors.Chainf(appErrors.ErrInternal, "%s.%s", uc.pkg, op)
	}

	for i := 0; i < count; i++ {
		newTaskID := uuid.NewString()

		payload := tworker.JobPayload{
			TaskID:     newTaskID,
			Filename:   filename,
			FileBucket: string(storage.BucketCommonFiles),
			FileKey:    fileKey,
		}

		opts := client.StartWorkflowOptions{
			ID:        "job-" + newTaskID,
			TaskQueue: tworker.TaskQueue,
		}

		_, err := uc.temporalClient.ExecuteWorkflow(ctx, opts, "JobWorkflow", payload)
		if err != nil {
			return fmt.Errorf("execute workflow: %w", err)
		}
	}

	return nil
}
