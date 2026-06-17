package types

import "errors"

var (
	// ErrNoMatch is returned by a NutritionSource or a Store food lookup when no
	// suitable food could be resolved. The pipeline treats it as a signal to try
	// the next configured source rather than as a failure.
	ErrNoMatch = errors.New("no food match")

	// ErrNotFound is returned by Store reads when the requested row is absent.
	ErrNotFound = errors.New("not found")
)
