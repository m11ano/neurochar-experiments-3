package fxboot

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app/config"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app/fxboot/invoking"
	"github.com/m11ano/neurochar-experiments-3/service/internal/app/fxboot/providing"
	"github.com/m11ano/neurochar-experiments-3/service/internal/domain/task"
	taskUC "github.com/m11ano/neurochar-experiments-3/service/internal/domain/task/usecase"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/db"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/storage"
	storageMigrations "github.com/m11ano/neurochar-experiments-3/service/internal/infra/storage/migrations"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/storage/s3d"
	temporalClient "github.com/m11ano/neurochar-experiments-3/service/internal/infra/temporal/client"
	temporalWorker "github.com/m11ano/neurochar-experiments-3/service/internal/infra/temporal/worker"
	"github.com/m11ano/neurochar-experiments-3/service/internal/tworker"
	"github.com/m11ano/neurochar-experiments-3/service/pkg/pgclient"
	tclient "go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func TemporalWorkerAppGetOptionsMap(appID app.ID, cfg config.Config) OptionsMap {
	return OptionsMap{
		Providing: map[ProvidingID]fx.Option{
			ProvidingAppID: fx.Provide(func() app.ID {
				return appID
			}),
			ProvidingIDFXTimeouts: fx.Options(
				fx.StartTimeout(time.Second*time.Duration(cfg.TemporalWorkerApp.Base.StartTimeoutSec)),
				fx.StopTimeout(time.Second*time.Duration(cfg.TemporalWorkerApp.Base.StopTimeoutSec)),
			),
			ProvidingIDConfig: fx.Provide(func() config.Config {
				return cfg
			}),
			ProvidingIDLogger: fx.Provide(func(cfg config.Config) *slog.Logger {
				return providing.NewLogger(
					cfg.TemporalWorkerApp.Name,
					cfg.TemporalWorkerApp.Version,
					cfg.TemporalWorkerApp.Base.UseLogger,
					cfg.TemporalWorkerApp.Base.IsProd,
				)
			}),
			ProvidingIDFXLogger: fx.WithLogger(func(cfg config.Config) fxevent.Logger {
				return providing.NewFXLogger(cfg.TemporalWorkerApp.Base.UseFxLogger)
			}),
			ProvidingIDTemporalWorker: fx.Provide(
				func(cfg config.Config, logger *slog.Logger) (temporalWorker.WorkerClient, error) {
					return temporalWorker.NewClient(
						cfg.TemporalWorkerApp.Temporal.Host,
						cfg.TemporalWorkerApp.Temporal.Namespace,
						logger,
					)
				},
			),
			ProvidingIDDBClients: fx.Provide(
				func(logger *slog.Logger, cfg config.Config, shutdown fx.Shutdowner) db.MasterClient {
					return providing.NewDBClients(
						cfg.Postgres.Master.DSN,
						cfg.TemporalWorkerApp.Base.LogSQLQueries,
						logger,
						shutdown,
					)
				},
			),
			ProvidingIDStorageClient: fx.Provide(providing.NewStorageClient),
			ProvidingIDTemporalClient: fx.Provide(func() temporalClient.Client {
				return nil
			}),
			ProvidingIDTask: task.FxModule,
		},
		Invokes: []fx.Option{
			fx.Invoke(TemporalWorkerAppInitInvoke),
		},
	}
}

type TemporalWorkerInvokeInput struct {
	fx.In

	LC             fx.Lifecycle
	Shutdowner     fx.Shutdowner
	Invokes        []invoking.InvokeInit `group:"InvokeInit"`
	Logger         *slog.Logger
	Cfg            config.Config
	TemporalWorker temporalWorker.WorkerClient
	DBMasterClient db.MasterClient
	S3Client       *s3.Client
	StorageClient  storage.Client
	TaskFacade     *taskUC.Facade
}

// TemporalWorkerAppInitInvoke - app init
func TemporalWorkerAppInitInvoke(
	in TemporalWorkerInvokeInput,
) {
	ctxWithCancel, cancel := context.WithCancel(context.Background())
	ctxForTemporal, temporalCancel := context.WithCancel(context.Background())

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

			_, err = in.TemporalWorker.CheckHealth(ctx, &tclient.CheckHealthRequest{})
			if err != nil {
				in.Logger.ErrorContext(ctx, "failed to connect to temporal", slog.Any("error", err))
				return err
			} else {
				in.Logger.InfoContext(ctx, "connected to temporal")
			}

			go func() {
				err := tworker.RunWorker(
					ctxWithCancel,
					in.TemporalWorker,
					tworker.MlWorkersConfig{
						Service:         in.Cfg.TemporalWorkerApp.MLWorker.Service,
						Readiness:        in.Cfg.TemporalWorkerApp.MLWorker.Readiness,
						FallbackService: in.Cfg.TemporalWorkerApp.MLWorker.FallbackService,
					},
					in.StorageClient,
					in.TaskFacade,
					in.Logger,
				)
				if err != nil {
					in.Logger.ErrorContext(ctx, "failed to run worker", slog.Any("error", err))
				}

				temporalCancel()
			}()

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

			cancel()

			<-ctxForTemporal.Done()
			time.Sleep(time.Second)
			in.TemporalWorker.Close()

			// Закрываем postgress
			in.DBMasterClient.Close()
			in.Logger.InfoContext(ctx, "closing db clients")

			return nil
		},
	})
}
