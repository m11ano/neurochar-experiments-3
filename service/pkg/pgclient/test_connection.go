package pgclient

import (
	"context"
	"log/slog"
	"time"
)

// TestConnection - test connection with postgres
func TestConnection(
	ctx context.Context,
	client Client,
	logger *slog.Logger,
	maxAttempt int,
	attemptSleepSeconds int,
) error {
	attemp := 1
	var err error
	for attemp <= maxAttempt {
		err = client.Pool().Ping(ctx)
		if err != nil {
			logger.Info(
				"failed to connect to Postgress",
				slog.String("serverID", client.ServerID()),
				slog.Int("attemp", attemp),
			)
			time.Sleep(time.Duration(attemptSleepSeconds) * time.Second)
			attemp++
			continue
		}
		break
	}
	if err != nil {
		return err
	}

	return nil
}
