package pgclient

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func setupClient(mockPool pgxmock.PgxPoolIface) *clientImpl {
	return &clientImpl{
		serverID:        "test-server",
		pool:            mockPool,
		defaultIsoLevel: ReadCommitted,
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestNewClient_InvalidDSN(t *testing.T) {
	t.Parallel()

	_, err := NewClient(context.Background(), "sid", "not-a-valid-dsn", NewClientOpts{})
	if err == nil || !strings.Contains(err.Error(), "unable to parse dsn") {
		t.Fatalf("expected DSN parse error, got %v", err)
	}
}

func TestNewClient_Defaults(t *testing.T) {
	t.Parallel()

	dsn := "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
	client, err := NewClient(context.Background(), "sid", dsn, NewClientOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.readOnly {
		t.Errorf("readOnly = %v, want false", client.readOnly)
	}

	if client.defaultIsoLevel != ReadCommitted {
		t.Errorf("defaultIsoLevel = %v, want %v", client.defaultIsoLevel, pgx.ReadCommitted)
	}

	if client.logger != nil {
		t.Errorf("logger = %v, want nil", client.logger)
	}

	cfg := client.pool.Config()
	if val, ok := cfg.ConnConfig.RuntimeParams["default_transaction_read_only"]; ok {
		t.Errorf("default_transaction_read_only = %q, want unset", val)
	}

	if cfg.ConnConfig.Tracer != nil {
		t.Errorf("Tracer = %T, want nil", cfg.ConnConfig.Tracer)
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	opts := NewClientOpts{
		ReadOnly:        true,
		LogQueries:      true,
		DefaultIsoLevel: Serializable,
		Logger:          logger,
	}
	dsn := "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
	client, err := NewClient(context.Background(), "sid", dsn, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !client.readOnly {
		t.Errorf("readOnly = %v, want true", client.readOnly)
	}

	if client.defaultIsoLevel != Serializable {
		t.Errorf("defaultIsoLevel = %v, want %v", client.defaultIsoLevel, Serializable)
	}

	if client.logger != logger {
		t.Errorf("logger = %v, want %v", client.logger, logger)
	}

	cfg := client.pool.Config()
	if val := cfg.ConnConfig.RuntimeParams["default_transaction_read_only"]; val != "true" {
		t.Errorf("default_transaction_read_only = %q, want \"true\"", val)
	}

	if cfg.ConnConfig.Tracer == nil {
		t.Errorf("Tracer = nil, want non-nil")
	}
}

func TestServerID(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mockPool.Close()

	client := setupClient(mockPool)
	if got := client.ServerID(); got != "test-server" {
		t.Errorf("ServerID() = %q, want %q", got, "test-server")
	}
}

func TestGetConn_NoTx(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	extra := client.GetConn(context.Background())
	if extra != mockPool {
		t.Errorf("GetConn without tx returned %T, want pool", extra)
	}
}

func TestDo_CommitsTransaction(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	mockPool.ExpectCommit()

	err := client.Do(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Do() unexpected error: %v", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDo_RollsBackOnError(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	mockPool.ExpectRollback()

	err := client.Do(context.Background(), func(ctx context.Context) error {
		return errors.New("boom")
	})
	if err == nil || err.Error() != "boom" {
		t.Errorf("Do() error = %v, want boom", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDoWithIsoLvl_CustomIsolation(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	iso := Serializable
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.TxIsoLevel(iso)})
	mockPool.ExpectCommit()

	err := client.DoWithIsoLvl(context.Background(), iso, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("DoWithIsoLvl() unexpected error: %v", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestNestedDo_Savepoints(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	mockPool.ExpectExec("SAVEPOINT sp_1").WillReturnResult(pgxmock.NewResult("SAVEPOINT", 1))
	mockPool.ExpectExec("RELEASE SAVEPOINT sp_1").WillReturnResult(pgxmock.NewResult("RELEASE", 1))
	mockPool.ExpectCommit()

	err := client.Do(context.Background(), func(ctx context.Context) error {
		return client.Do(ctx, func(ctx context.Context) error {
			return nil
		})
	})
	if err != nil {
		t.Errorf("Nested Do() unexpected error: %v", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetConn_WithTx(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("unable to create mock pool: %v", err)
	}
	defer mockPool.Close()

	client := setupClient(mockPool)

	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	tx, err := mockPool.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	ctx := context.WithValue(context.Background(), client.txKey, tx)
	conn := client.GetConn(ctx)

	if conn != tx {
		t.Errorf("GetConn(ctx) = %T, want %T", conn, tx)
	}

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unmet expectations: %v", err)
	}
}

func TestPool_ReturnsUnderlyingPool(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	if got := client.Pool(); got != mockPool {
		t.Errorf("Pool() = %T, want %T", got, mockPool)
	}
}

func TestDoWithIsoLvl_RollsBackToSavepointOnError(t *testing.T) {
	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	mockPool.ExpectExec("SAVEPOINT sp_1").WillReturnResult(pgxmock.NewResult("SAVEPOINT", 1))
	mockPool.ExpectExec("ROLLBACK TO SAVEPOINT sp_1").WillReturnResult(pgxmock.NewResult("ROLLBACK", 1))
	mockPool.ExpectRollback()

	err := client.Do(context.Background(), func(ctx context.Context) error {
		return client.DoWithIsoLvl(ctx, ReadCommitted, func(ctx context.Context) error {
			return errors.New("inner fail")
		})
	})
	if err == nil || !strings.Contains(err.Error(), "inner fail") {
		t.Errorf("expected inner error, got %v", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDoWithIsoLvl_NestedReleaseOnSuccess(t *testing.T) {
	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	mockPool.ExpectExec("SAVEPOINT sp_1").WillReturnResult(pgxmock.NewResult("SAVEPOINT", 1))
	mockPool.ExpectExec("RELEASE SAVEPOINT sp_1").WillReturnResult(pgxmock.NewResult("RELEASE", 1))
	mockPool.ExpectCommit()

	err := client.Do(context.Background(), func(ctx context.Context) error {
		return client.DoWithIsoLvl(ctx, ReadCommitted, func(ctx context.Context) error {
			return nil
		})
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDoWithIsoLvl_PanicRollsBack(t *testing.T) {
	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	mockPool.ExpectRollback()

	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()
		_ = client.DoWithIsoLvl(context.Background(), ReadCommitted, func(ctx context.Context) error {
			panic("oops")
		})
	}()
	if !didPanic {
		t.Errorf("expected panic, but none occurred")
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDoWithIsoLvl_BeginTxError(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	err := client.DoWithIsoLvl(context.Background(), ReadCommitted, func(ctx context.Context) error {
		return nil
	})
	if err == nil {
		t.Errorf("expected BeginTx error, got nil")
	}
}

func TestDoWithIsoLvl_SavepointExecError(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	spErr := errors.New("savepoint fail")
	mockPool.ExpectExec("SAVEPOINT sp_1").WillReturnError(spErr)
	mockPool.ExpectRollback()

	err := client.DoWithIsoLvl(context.Background(), ReadCommitted, func(ctx context.Context) error {
		return client.DoWithIsoLvl(ctx, ReadCommitted, func(ctx context.Context) error {
			return nil
		})
	})
	if err != spErr {
		t.Errorf("expected savepoint error, got %v", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDoWithIsoLvl_ReleaseExecError(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	mockPool.ExpectExec("SAVEPOINT sp_1").WillReturnResult(pgxmock.NewResult("SAVEPOINT", 1))
	mockPool.ExpectExec("RELEASE SAVEPOINT sp_1").WillReturnError(errors.New("release fail"))
	mockPool.ExpectRollback()

	err := client.DoWithIsoLvl(context.Background(), ReadCommitted, func(ctx context.Context) error {
		return client.DoWithIsoLvl(ctx, ReadCommitted, func(ctx context.Context) error {
			return nil
		})
	})
	if err == nil || err.Error() != "release fail" {
		t.Errorf("expected release savepoint error, got %v", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDoWithIsoLvl_RollbackSavepointErrorLogButOriginalReturned(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	mockPool.ExpectExec("SAVEPOINT sp_1").WillReturnResult(pgxmock.NewResult("SAVEPOINT", 1))
	mockPool.ExpectExec("ROLLBACK TO SAVEPOINT sp_1").WillReturnError(errors.New("rb fail"))
	mockPool.ExpectRollback().WillReturnError(errors.New("rb fail"))

	err := client.DoWithIsoLvl(context.Background(), ReadCommitted, func(ctx context.Context) error {
		return client.DoWithIsoLvl(ctx, ReadCommitted, func(ctx context.Context) error {
			return errors.New("inner error")
		})
	})
	if err == nil || err.Error() != "inner error" {
		t.Errorf("expected original inner error, got %v", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDo_CommitError(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	mockPool.ExpectCommit().WillReturnError(errors.New("commit failed"))

	err := client.Do(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err == nil || err.Error() != "commit failed" {
		t.Errorf("Do() error = %v, want commit failed", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDo_NestedCommitSavepoint(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	mockPool.ExpectExec("SAVEPOINT sp_1").WillReturnResult(pgxmock.NewResult("SAVEPOINT", 0))
	mockPool.ExpectExec("RELEASE SAVEPOINT sp_1").WillReturnResult(pgxmock.NewResult("RELEASE", 0))
	mockPool.ExpectCommit()

	err := client.Do(context.Background(), func(ctx context.Context) error {
		return client.Do(ctx, func(ctx context.Context) error {
			return nil
		})
	})
	if err != nil {
		t.Errorf("nested Do() unexpected error: %v", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDo_PanicRollback(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	mockPool.ExpectBeginTx(pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	mockPool.ExpectRollback().WillReturnError(errors.New("rb fail"))

	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()
		_ = client.Do(context.Background(), func(ctx context.Context) error {
			panic("panic")
		})
	}()

	if !didPanic {
		t.Errorf("expected panic to propagate")
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestClose(t *testing.T) {
	t.Parallel()

	mockPool, _ := pgxmock.NewPool()
	defer mockPool.Close()

	client := setupClient(mockPool)
	client.Close()
}
