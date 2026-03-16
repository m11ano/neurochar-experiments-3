// Package fxboot contains fx bootstrapping
package fxboot

import (
	"go.uber.org/fx"
)

// ProvidingID - type for providing id
type ProvidingID int

const (
	// ProvidingAppID - app id
	ProvidingAppID ProvidingID = iota

	// ProvidingIDFXTimeouts - fx timeouts
	ProvidingIDFXTimeouts

	// ProvidingIDConfig - app config
	ProvidingIDConfig

	// ProvidingIDLogger - logger
	ProvidingIDLogger

	// ProvidingIDFXLogger - fx logger
	ProvidingIDFXLogger

	// ProvidingIDDBClients - db clients
	ProvidingIDDBClients

	// ProvidingIDStorageClient - storage client
	ProvidingIDStorageClient

	// ProvidingHTTPFiberServer - http fiber server
	ProvidingHTTPFiberServer

	// ProvidingIDDeliveryHTTP - delivery http
	ProvidingIDDeliveryHTTP

	// ProvidingIDTask - task
	ProvidingIDTask

	// ProvidingIDTemporalWorker - temporal worker
	ProvidingIDTemporalWorker

	// ProvidingIDTemporalClient - temporal client
	ProvidingIDTemporalClient
)

// OptionsMap - options map item with providing and invokes elements
type OptionsMap struct {
	Providing map[ProvidingID]fx.Option
	Invokes   []fx.Option
}

// OptionsMapToSlice - convert options map to slice for fx bootstrapping
func OptionsMapToSlice(optionsMap OptionsMap) []fx.Option {
	result := make([]fx.Option, 0)

	for _, option := range optionsMap.Providing {
		result = append(result, option)
	}

	result = append(result, optionsMap.Invokes...)

	return result
}
