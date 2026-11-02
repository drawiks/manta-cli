package manta

import "errors"

var (
	// ErrNotReplay reports that the input does not start with the Source 2
	// replay magic.
	ErrNotReplay = errors.New("not a Dota 2 replay")
	// ErrNoPlayerResource reports that a replay has no CDOTA_PlayerResource.
	ErrNoPlayerResource = errors.New("CDOTA_PlayerResource not found")
)
