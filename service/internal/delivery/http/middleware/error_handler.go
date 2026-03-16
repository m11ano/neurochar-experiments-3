// Package middleware contains middleware for http handlers
package middleware

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	appErrors "github.com/m11ano/neurochar-experiments-3/service/internal/app/errors"
)

type errorJSON struct {
	Code     int            `json:"code"`
	TextCode string         `json:"textCode"`
	Hints    []string       `json:"hints"`
	Details  map[string]any `json:"details"`
}

// ErrorHandler - обработчик ошибок
func ErrorHandler(appTitle string, logger *slog.Logger) func(*fiber.Ctx, error) error {
	return func(c *fiber.Ctx, err error) error {
		code := 500
		jsonRes := errorJSON{
			TextCode: "INTERNAL_ERROR",
			Hints:    []string{},
			Details:  map[string]any{},
		}

		if appError, ok := appErrors.ExtractError(err); ok {
			code = int(appError.Meta().Code)
			jsonRes.TextCode = appError.Meta().TextCode
			jsonRes.Hints = appError.Hints()
			jsonRes.Details = appError.Details(false)
		} else {
			switch errTyped := err.(type) {
			case *fiber.Error:
				code = errTyped.Code
				switch {
				case code == 405:
					jsonRes.TextCode = "METHOD_NOT_ALLOWED"
				case code >= 400 && code < 500:
					jsonRes.TextCode = "BAD_REQUEST"
				}
				jsonRes.Hints = []string{errTyped.Message}
			default:
			}
		}

		jsonRes.Code = code

		return c.Status(code).JSON(jsonRes)
	}
}
