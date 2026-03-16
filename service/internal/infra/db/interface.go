// Package db contains db interfaces and helpers
package db

import "github.com/m11ano/neurochar-experiments-3/service/pkg/pgclient"

// MasterClient - interface for master (read + write)
type MasterClient interface {
	pgclient.Client
}
