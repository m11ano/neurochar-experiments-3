package client

import (
	"log/slog"

	tclient "go.temporal.io/sdk/client"
)

type Client tclient.Client

func NewClient(endpoint string, namespace string, logger *slog.Logger) (Client, error) {
	c, err := tclient.NewLazyClient(tclient.Options{
		HostPort:  endpoint,
		Logger:    logger.WithGroup("temporal_client"),
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}
