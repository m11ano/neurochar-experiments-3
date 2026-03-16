package main

import (
	"github.com/m11ano/neurochar-experiments-3/service/internal/app"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app/config"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app/fxboot"
	"go.uber.org/fx"
)

func main() {
	cfg := config.LoadConfig("configs/base.yml", "configs/base.local.yml")

	appOptions := fxboot.TemporalWorkerAppGetOptionsMap(app.IDTemporalWorker, cfg)

	app := fx.New(
		fxboot.OptionsMapToSlice(appOptions)...,
	)

	app.Run()
}
