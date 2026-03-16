package db

import (
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// UpMigrations - up goose migrations
func UpMigrations(dsn string, migrationsPath string, logger *slog.Logger) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("unable to set dialect: %w", err)
	}

	gooseLogger := &migrationsLogger{logger: logger}
	goose.SetLogger(gooseLogger)

	pgxCfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("unable to parse dsn: %w", err)
	}

	db := stdlib.OpenDB(*pgxCfg)
	defer func() {
		err := db.Close()
		if err != nil {
			logger.Error(fmt.Sprintf("unable to close db connection: %v", err))
		}
	}()

	if err := goose.Up(db, migrationsPath); err != nil {
		return fmt.Errorf("unable to run up migrations: %w", err)
	}

	return nil
}

// DownMigrations - down goose migrations
func DownMigrations(dsn string, migrationsPath string, logger *slog.Logger) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("unable to set dialect: %w", err)
	}

	gooseLogger := &migrationsLogger{logger: logger}
	goose.SetLogger(gooseLogger)

	pgxCfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("unable to parse dsn: %w", err)
	}

	db := stdlib.OpenDB(*pgxCfg)
	defer func() {
		err := db.Close()
		if err != nil {
			logger.Error(fmt.Sprintf("unable to close db connection: %v", err))
		}
	}()

	if err := goose.DownTo(db, migrationsPath, 0); err != nil {
		return fmt.Errorf("unable to run down migrations: %w", err)
	}

	return nil
}

type migrationsLogger struct {
	logger *slog.Logger
}

func (l *migrationsLogger) Fatalf(format string, v ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, v...))
}

func (l *migrationsLogger) Printf(format string, v ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, v...))
}
