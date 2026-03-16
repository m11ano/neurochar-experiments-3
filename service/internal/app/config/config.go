package config

import (
	"log"
	"log/slog"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config struct for app config
type Config struct {
	BackendApp struct {
		Name    string `yaml:"name" env:"BACKEND_APP_NAME" env-default:"backend"`
		Version string `yaml:"version" env:"BACKEND_APP_VERSION" env-default:"1.0.0"`
		Base    struct {
			StartTimeoutSec int  `yaml:"start_timeout_sec" env:"BACKEND_APP_BASE_START_TIMEOUT_SEC" env-default:"10"`
			StopTimeoutSec  int  `yaml:"stop_timeout_sec" env:"BACKEND_APP_BASE_STOP_TIMEOUT_SEC" env-default:"2"`
			IsProd          bool `yaml:"is_prod" env:"BACKEND_APP_BASE_IS_PROD" env-default:"false"`
			UseFxLogger     bool `yaml:"use_fx_logger" env:"BACKEND_APP_BASE_USE_FX_LOGGER"`
			UseLogger       bool `yaml:"use_logger" env:"BACKEND_APP_BASE_USE_LOGGER"`
			LogSQLQueries   bool `yaml:"log_sql_queries" env:"BACKEND_APP_BASE_LOG_SQL_QUERIES"`
			LogHTTP         bool `yaml:"log_http" env:"BACKEND_APP_BASE_LOG_HTTP"`
		} `yaml:"base"`
		HTTP struct {
			Port             int      `yaml:"port" env:"BACKEND_APP_HTTP_PORT" env-default:"8080"`
			Prefix           string   `yaml:"prefix" env:"BACKEND_APP_HTTP_PREFIX" env-default:""`
			UnderProxy       bool     `yaml:"under_proxy" env:"BACKEND_APP_HTTP_UNDER_PROXY" env-default:"false"`
			StopTimeoutSec   int      `yaml:"stop_timeout_sec" env:"BACKEND_APP_HTTP_STOP_TIMEOUT_SEC" env-default:"3"`
			CorsAllowOrigins []string `yaml:"cors_allow_origins" env:"BACKEND_APP_HTTP_CORS_ALLOW_ORIGINS" env-default:""`
		} `yaml:"http"`
	} `yaml:"backend_app"`
	TemporalWorkerApp struct {
		Name    string `yaml:"name" env:"TEMPORAL_WORKER_APP_NAME" env-default:"temporal_worker"`
		Version string `yaml:"version" env:"TEMPORAL_WORKER_APP_VERSION" env-default:"1.0.0"`
		Base    struct {
			StartTimeoutSec int  `yaml:"start_timeout_sec" env:"TEMPORAL_WORKER_APP_BASE_START_TIMEOUT_SEC" env-default:"10"`
			StopTimeoutSec  int  `yaml:"stop_timeout_sec" env:"TEMPORAL_WORKER_APP_BASE_STOP_TIMEOUT_SEC" env-default:"2"`
			IsProd          bool `yaml:"is_prod" env:"TEMPORAL_WORKER_APP_BASE_IS_PROD"`
			UseFxLogger     bool `yaml:"use_fx_logger" env:"TEMPORAL_WORKER_APP_BASE_USE_FX_LOGGER"`
			UseLogger       bool `yaml:"use_logger" env:"TEMPORAL_WORKER_APP_BASE_USE_LOGGER"`
			LogSQLQueries   bool `yaml:"log_sql_queries" env:"TEMPORAL_WORKER_APP_BASE_LOG_SQL_QUERIES"`
		} `yaml:"base"`
		Temporal struct {
			Host      string `yaml:"host" env:"TEMPORAL_HOST" env-default:"127.0.0.1:7233"`
			Namespace string `yaml:"namespace" env:"TEMPORAL_NAMESPACE" env-default:"default"`
		} `yaml:"temporal"`
		MLWorker struct {
			Service         string `yaml:"service" env:"TEMPORAL_WORKER_APP_ML_WORKER_SERVICE"`
			Readiness       string `yaml:"readiness" env:"TEMPORAL_WORKER_APP_ML_WORKER_READINESS"`
			FallbackService string `yaml:"fallback_service" env:"TEMPORAL_WORKER_APP_ML_WORKER_FALLBACK_SERVICE"`
		} `yaml:"ml_worker"`
	} `yaml:"temporal_worker_app"`
	Postgres struct {
		MaxAttempts         int    `yaml:"max_attempts" env:"POSTGRES_MAX_ATTEMPTS" env-default:"3"`
		AttemptSleepSeconds int    `yaml:"attempt_sleep_seconds" env:"POSTGRES_ATTEMPT_SLEEP_SECONDS" env-default:"1"`
		MigrationsPath      string `yaml:"migrations_path" env:"POSTGRES_MIGRATIONS_PATH" env-default:"migrations"`
		Master              struct {
			DSN string `yaml:"dsn" env:"POSTGRES_MASTER_DSN"`
		} `yaml:"master"`
	} `yaml:"postgres"`
	Storage struct {
		UpMigrations     bool   `yaml:"up_migrations" env:"STORAGE_UP_MIGRATIONS" env-default:"false"`
		S3Endpoint       string `yaml:"s3_endpoint" env:"STORAGE_S3_ENDPOINT" env-default:""`
		S3AccessKey      string `yaml:"s3_access_key" env:"STORAGE_S3_ACCESS_KEY" env-default:""`
		S3SecretKey      string `yaml:"s3_secret_key" env:"STORAGE_S3_SECRET_KEY" env-default:""`
		S3Region         string `yaml:"s3_region" env:"STORAGE_S3_REGION" env-default:""`
		S3URL            string `yaml:"s3_url" env:"STORAGE_S3_URL" env-default:""`
		S3URLIsHost      bool   `yaml:"s3_url_is_host" env:"STORAGE_S3_URL_IS_HOST" env-default:"false"`
		S3URLHostPrefix  string `yaml:"s3_url_host_prefix" env:"STORAGE_S3_URL_HOST_PREFIX" env-default:""`
		S3URLHostPostfix string `yaml:"s3_url_host_postfix" env:"STORAGE_S3_URL_HOST_POSTFIX" env-default:""`
	} `yaml:"storage"`
}

// LoadConfig loads app config from file
func LoadConfig(files ...string) Config {
	var Config Config

	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			err := cleanenv.ReadConfig(file, &Config)
			if err != nil {
				log.Fatal("config file error", err)
			}
		} else {
			slog.Warn("config file not found", slog.String("file", file))
		}
	}

	return Config
}
