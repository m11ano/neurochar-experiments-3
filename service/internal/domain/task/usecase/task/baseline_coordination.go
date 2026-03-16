package task

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	appErrors "github.com/m11ano/neurochar-experiments-3/service/internal/app/errors"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/entity"
)

var instances = []string{
	"http://127.0.0.1:8000/extract-text-ocr",
	"http://127.0.0.1:8001/extract-text-ocr",
	"http://127.0.0.1:8002/extract-text-ocr",
	"http://127.0.0.1:8003/extract-text-ocr",
}

func (uc *UsecaseImpl) StartTaskWithBaselineCoordination(
	ctx context.Context,
	filename string,
	filedata []byte,
	count int,
) error {
	const op = "StartTaskWithBaselineCoordination"

	for i := 0; i < count; i++ {
		err := uc.workerpool.Submit(ctx, func(ctx context.Context, workerNo int) error {
			task, err := entity.NewTask(filename, "baseline")
			if err != nil {
				return err
			}

			err = uc.repo.Create(ctx, task)
			if err != nil {
				return err
			}

			workerURL := instances[workerNo-1]

			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)

			part, err := writer.CreateFormFile("file", filename)
			if err != nil {
				return err
			}

			if _, err := part.Write(filedata); err != nil {
				return err
			}

			if err := writer.Close(); err != nil {
				return err
			}

			req, err := http.NewRequestWithContext(
				ctx,
				http.MethodPost,
				workerURL,
				&buf,
			)
			if err != nil {
				return err
			}

			req.Header.Set("Content-Type", writer.FormDataContentType())

			client := &http.Client{
				Timeout: 5 * time.Minute,
			}

			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf(
					"worker %d (%s) status %d, body: %s",
					workerNo,
					workerURL,
					resp.StatusCode,
					string(body),
				)
			}

			task.IsProcessed = true
			task.Result = string(body)

			err = uc.repo.Update(ctx, task)
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return appErrors.Chainf(err, "%s.%s", uc.pkg, op)
		}
	}

	return nil
}
