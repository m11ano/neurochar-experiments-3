package pgclient

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestConnection using real clientImpl backed by pgxmock
func TestConnection_SuccessFirstTry(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mockPool.Close()

	client := setupClient(mockPool)
	// Ping should succeed immediately
	mockPool.ExpectPing()

	err = TestConnection(context.Background(), client, slog.New(slog.NewTextHandler(io.Discard, nil)), 3, 0)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestConnection_RetriesThenSucceeds(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mockPool.Close()

	client := setupClient(mockPool)
	// First two pings fail, third succeeds
	err1 := errors.New("fail1")
	err2 := errors.New("fail2")
	mockPool.ExpectPing().WillReturnError(err1)
	mockPool.ExpectPing().WillReturnError(err2)
	mockPool.ExpectPing()

	err = TestConnection(context.Background(), client, slog.New(slog.NewTextHandler(io.Discard, nil)), 5, 0)
	if err != nil {
		t.Errorf("expected no error after retries, got %v", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestConnection_FailsAfterMaxAttempts(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mockPool.Close()

	client := setupClient(mockPool)
	// Two pings fail, maxAttempt=2
	err1 := errors.New("fail1")
	err2 := errors.New("fail2")
	mockPool.ExpectPing().WillReturnError(err1)
	mockPool.ExpectPing().WillReturnError(err2)

	err = TestConnection(context.Background(), client, slog.New(slog.NewTextHandler(io.Discard, nil)), 2, 0)
	if err == nil || err.Error() != err2.Error() {
		t.Errorf("expected error '%v', got %v", err2, err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestConnection_ZeroMaxAttempt(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mockPool.Close()

	client := setupClient(mockPool)
	// maxAttempt=0: no ping calls
	err = TestConnection(context.Background(), client, slog.New(slog.NewTextHandler(io.Discard, nil)), 0, 0)
	if err != nil {
		t.Errorf("expected no error when maxAttempt is zero, got %v", err)
	}
	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected ping attempts: %v", err)
	}
}
