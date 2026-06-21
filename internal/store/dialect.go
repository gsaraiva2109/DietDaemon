// Package store implements ports.Store with SQLite via a pure-Go driver.
package store

// Dialect abstracts SQL dialect differences between SQLite and Postgres.
// Currently constructs the SQLite variant; Postgres is reserved for
// future phases.
type Dialect interface {
	// Placeholder returns the parameter placeholder for the n-th argument
	// (1-based): "?" for SQLite, "$n" for Postgres.
	Placeholder(n int) string
}

// sqliteDialect returns SQLite placeholders ("?").
type sqliteDialect struct{}

func (d sqliteDialect) Placeholder(int) string { return "?" }

// NewDialect returns the dialect for the given driver. Currently
// supports "sqlite".
func NewDialect(driver string) (Dialect, error) {
	if driver != "sqlite" {
		return nil, &ErrUnsupportedDriver{Driver: driver}
	}
	return sqliteDialect{}, nil
}

// ErrUnsupportedDriver is returned when DB_DRIVER is not "sqlite".
type ErrUnsupportedDriver struct{ Driver string }

func (e *ErrUnsupportedDriver) Error() string {
	return "DB_DRIVER=" + e.Driver + " not supported yet; only 'sqlite'"
}
