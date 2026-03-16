package fxboot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v2"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app/config"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app/fxboot/invoking"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app/fxboot/providing"
	backendHTTP "github.com/m11ano/neurochar-experiments-3/service/internal/delivery/http/backend"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/db"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/storage"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/storage/s3d"
	temporalClient "github.com/m11ano/neurochar-experiments-3/service/internal/infra/temporal/client"
	"github.com/m11ano/neurochar-experiments-3/service/pkg/pgclient"
	tclient "go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	storageMigrations "github.com/m11ano/neurochar-experiments-3/service/internal/infra/storage/migrations"
)

func BackendAppGetOptionsMap(appID app.ID, cfg config.Config) OptionsMap {
	return OptionsMap{
		Providing: map[ProvidingID]fx.Option{
			ProvidingAppID: fx.Provide(func() app.ID {
				return appID
			}),
			ProvidingIDFXTimeouts: fx.Options(
				fx.StartTimeout(time.Second*time.Duration(cfg.BackendApp.Base.StartTimeoutSec)),
				fx.StopTimeout(time.Second*time.Duration(cfg.BackendApp.Base.StopTimeoutSec)),
			),
			ProvidingIDConfig: fx.Provide(func() config.Config {
				return cfg
			}),
			ProvidingIDLogger: fx.Provide(func(cfg config.Config) *slog.Logger {
				return providing.NewLogger(
					cfg.BackendApp.Name,
					cfg.BackendApp.Version,
					cfg.BackendApp.Base.UseLogger,
					cfg.BackendApp.Base.IsProd,
				)
			}),
			ProvidingIDFXLogger: fx.WithLogger(func(cfg config.Config) fxevent.Logger {
				return providing.NewFXLogger(cfg.BackendApp.Base.UseFxLogger)
			}),
			ProvidingIDDBClients: fx.Provide(
				func(logger *slog.Logger, cfg config.Config, shutdown fx.Shutdowner) db.MasterClient {
					return providing.NewDBClients(
						cfg.Postgres.Master.DSN,
						cfg.BackendApp.Base.LogSQLQueries,
						logger,
						shutdown,
					)
				},
			),
			ProvidingIDStorageClient: fx.Provide(providing.NewStorageClient),
			ProvidingHTTPFiberServer: fx.Provide(
				func(logger *slog.Logger, cfg config.Config) *fiber.App {
					httpConfig := backendHTTP.HTTPConfig{
						AppTitle:         "Neurochar Experiments 1",
						UnderProxy:       cfg.BackendApp.HTTP.UnderProxy,
						UseLogger:        cfg.BackendApp.Base.UseLogger && cfg.BackendApp.Base.LogHTTP,
						BodyLimit:        -1,
						CorsAllowOrigins: cfg.BackendApp.HTTP.CorsAllowOrigins,
						ServerIPs:        []string{"127.0.0.1"},
					}

					return backendHTTP.NewHTTPFiber(httpConfig, logger)
				},
			),
			ProvidingIDDeliveryHTTP: backendHTTP.FxModule,
			ProvidingIDTask:         task.FxModule,
			ProvidingIDTemporalClient: fx.Provide(func(cfg config.Config, logger *slog.Logger) (temporalClient.Client, error) {
				return temporalClient.NewClient(
					cfg.TemporalWorkerApp.Temporal.Host,
					cfg.TemporalWorkerApp.Temporal.Namespace,
					logger,
				)
			}),
		},
		Invokes: []fx.Option{
			fx.Invoke(BackendAppInitInvoke),
		},
	}
}

type BAckendInvokeInput struct {
	fx.In

	LC              fx.Lifecycle
	Shutdowner      fx.Shutdowner
	Invokes         []invoking.InvokeInit `group:"InvokeInit"`
	Logger          *slog.Logger
	Cfg             config.Config
	DBMasterClient  db.MasterClient
	S3Client        *s3.Client
	StorageClient   storage.Client
	HttpFiberServer *fiber.App
	TemporalClient  temporalClient.Client
}

// BackendAppInitInvoke - app init
func BackendAppInitInvoke(
	in BAckendInvokeInput,
) {
	in.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Тестирование соединения с мастером postgress
			err := pgclient.TestConnection(
				ctx,
				in.DBMasterClient,
				in.Logger,
				in.Cfg.Postgres.MaxAttempts,
				in.Cfg.Postgres.AttemptSleepSeconds,
			)
			if err != nil {
				in.Logger.ErrorContext(ctx, "failed to test master db connection", slog.Any("error", err))
				return err
			}

			in.Logger.InfoContext(
				ctx,
				"successfully connected to Postgress",
				slog.String("serverID", in.DBMasterClient.ServerID()),
			)

			// Миграции goose
			err = db.UpMigrations(in.Cfg.Postgres.Master.DSN, in.Cfg.Postgres.MigrationsPath, in.Logger)
			if err != nil {
				in.Logger.ErrorContext(ctx, "failed to run migrations", slog.Any("error", err))
				return err
			}

			// Тестирование соединения с s3
			err = s3d.PingS3Client(ctx, in.S3Client)
			if err != nil {
				in.Logger.ErrorContext(ctx, "failed to ping s3", slog.Any("error", err))
				return err
			}
			in.Logger.InfoContext(ctx, "connected to s3")

			// Миграции хранилища
			if in.Cfg.Storage.UpMigrations {
				createdAny, err := storageMigrations.UpBuckets(ctx, in.StorageClient)
				if err != nil {
					in.Logger.ErrorContext(ctx, "failed to migrate storage", slog.Any("error", err))
					return err
				}

				if createdAny {
					in.Logger.InfoContext(ctx, "storage buckets created")
				} else {
					in.Logger.InfoContext(ctx, "storage buckets already exist")
				}
			} else {
				in.Logger.InfoContext(ctx, "storage migrations skipped")
			}

			_, err = in.TemporalClient.CheckHealth(ctx, &tclient.CheckHealthRequest{})
			if err != nil {
				in.Logger.ErrorContext(ctx, "failed to connect to temporal", slog.Any("error", err))
				return err
			} else {
				in.Logger.InfoContext(ctx, "connected to temporal")
			}

			// Запускаем invoke функции до открытия
			for _, invokeItem := range in.Invokes {
				if invokeItem.StartBeforeOpen != nil {
					err := invokeItem.StartBeforeOpen(ctx)
					if err != nil {
						in.Logger.ErrorContext(ctx, "failed to execute invoke fn start before open", slog.Any("error", err))
						return err
					}
				}
			}

			// Запускаем http
			if in.Cfg.BackendApp.HTTP.Port > 0 {
				in.Logger.InfoContext(ctx, "starting http server", slog.Int("port", in.Cfg.BackendApp.HTTP.Port))
				go func() {
					if err := in.HttpFiberServer.Listen(fmt.Sprintf(":%d", in.Cfg.BackendApp.HTTP.Port)); err != nil {
						in.Logger.ErrorContext(ctx, "failed to start fiber", slog.Any("error", err))
						err := in.Shutdowner.Shutdown()
						if err != nil {
							in.Logger.ErrorContext(ctx, "failed to shutdown", slog.Any("error", err))
						}
					}
				}()
			}

			// Запускаем invoke функции после открытия
			for _, invokeItem := range in.Invokes {
				if invokeItem.StartAfterOpen != nil {
					err := invokeItem.StartAfterOpen(ctx)
					if err != nil {
						in.Logger.ErrorContext(ctx, "failed to execute invoke fn start after open", slog.Any("error", err))
						return err
					}
				}
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			for _, invokeItem := range in.Invokes {
				if invokeItem.Stop != nil {
					err := invokeItem.Stop(ctx)
					if err != nil {
						in.Logger.ErrorContext(ctx, "failed to execute invoke fn stop", slog.Any("error", err))
						return err
					}
				}
			}

			// Останавливаем http
			if in.Cfg.BackendApp.HTTP.Port > 0 {
				in.Logger.Info("stopping http fiber")
				err := in.HttpFiberServer.ShutdownWithTimeout(time.Duration(in.Cfg.BackendApp.HTTP.StopTimeoutSec) * time.Second)
				if err != nil {
					in.Logger.ErrorContext(ctx, "failed to stop fiber", slog.Any("error", err))
				}
			}

			// Закрываем postgress
			in.DBMasterClient.Close()
			in.Logger.InfoContext(ctx, "closing db clients")

			return nil
		},
	})
}
