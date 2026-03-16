package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	appErrors "github.com/m11ano/neurochar-experiments-3/service/internal/app/errors"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/loghandler"
)

// RequestData - request data
type RequestData struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	URI     string `json:"uri"`
	Referer string `json:"referer,omitempty"`
	IPChain string `json:"ip_chain"`
}

// ResponseData - response data
type ResponseData struct {
	DurationMS int64                 `json:"duration_ms"`
	Code       int                   `json:"code"`
	AppError   *appErrors.JSONStruct `json:"app_error,omitempty"`
	Error      error                 `json:"error,omitempty"`
}

// Logger - middleware for logging
func Logger(logger *slog.Logger) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		reqData := RequestData{
			Method:  c.Method(),
			Path:    c.Path(),
			URI:     string(c.Context().URI().QueryString()),
			Referer: c.Get("X-Referer"),
			IPChain: c.IP(),
		}

		ctxData, ctxKey := loghandler.SetData(c.Context(), "http.request", reqData)
		c.Locals(ctxKey, ctxData)

		start := time.Now()

		errResult := c.Next()

		duration := time.Since(start)

		if errResult != nil {
			err := c.App().ErrorHandler(c, errResult)
			if err != nil {
				logger.ErrorContext(
					loghandler.WithSource(c.Context()),
					"failed to call fiber error handler",
					slog.Any("error", err),
				)
			}
		}

		code := c.Response().StatusCode()

		resData := ResponseData{
			DurationMS: duration.Milliseconds(),
			Code:       code,
		}

		if errResult != nil {
			resData.Error = errResult
			errStr := appErrors.ToJSONStruct(errResult, true, false)
			resData.AppError = &errStr
		}

		ctx := loghandler.SetContextData(c.Context(), "http.response", resData)

		if code >= 400 {
			logger.ErrorContext(ctx, "http")
		} else {
			logger.InfoContext(ctx, "http")
		}

		return nil
	}
}
