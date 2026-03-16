package backend

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/m11ano/neurochar-experiments-3/service/internal/delivery/http/middleware"
)

const defaultBodyLimit = 50 * 1024 * 1024

// HTTPConfig - config for http server
type HTTPConfig struct {
	AppTitle         string
	UnderProxy       bool
	UseLogger        bool
	BodyLimit        int
	CorsAllowOrigins []string
	ServerIPs        []string
}

// NewHTTPFiber provides fiber app
func NewHTTPFiber(httpCfg HTTPConfig, logger *slog.Logger) *fiber.App {
	if httpCfg.BodyLimit == -1 {
		httpCfg.BodyLimit = defaultBodyLimit
	}

	fiberCfg := fiber.Config{
		ErrorHandler: middleware.ErrorHandler(httpCfg.AppTitle, logger),
		BodyLimit:    httpCfg.BodyLimit,
	}

	if httpCfg.UnderProxy {
		fiberCfg.ProxyHeader = fiber.HeaderXForwardedFor
	}

	app := fiber.New(fiberCfg)

	app.Use(middleware.Recovery(logger))
	app.Use(middleware.RequestID())
	app.Use(middleware.RequestIP(httpCfg.ServerIPs))

	if len(httpCfg.CorsAllowOrigins) > 0 {
		app.Use(middleware.Cors(httpCfg.CorsAllowOrigins))
	}

	if httpCfg.UseLogger {
		app.Use(middleware.Logger(logger))
	}

	return app
}
