// Package v1 contains v1 http handlers
package v1

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app/config"
)

type Groups struct {
	Prefix  string
	Default fiber.Router
}

const BackoffDefaultGroupID = "default"

// ProvideGroups - provide v1 group
func ProvideGroups(
	cfg config.Config,
	fiberApp *fiber.App,
) *Groups {
	prefix := fmt.Sprintf("%s/v1", cfg.BackendApp.HTTP.Prefix)

	defaultGroup := fiberApp.Group(prefix)

	return &Groups{
		Prefix:  prefix,
		Default: defaultGroup,
	}
}
