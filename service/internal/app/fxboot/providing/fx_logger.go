// Package providing contains fx bootstrapping for infrastructure
package providing

import (
	"os"

	"go.uber.org/fx/fxevent"
)

// NewFXLogger provides logger for fx
func NewFXLogger(useLogger bool) fxevent.Logger {
	if !useLogger {
		return fxevent.NopLogger
	}
	return &fxevent.ConsoleLogger{
		W: os.Stdout,
	}
}
