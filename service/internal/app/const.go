// Package app contains app constants
package app

// ID - app id
type ID int

const (
	// IDBackend - backend app id
	IDBackend ID = iota

	// IDTemporalWorker - temporal worker app id
	IDTemporalWorker
)
