package config

import "time"

const (
	// Connection pool bounds. Defaults are unlimited open connections, which lets
	// a traffic spike exhaust Postgres' own max_connections; idle/lifetime caps
	// keep the pool from holding stale connections indefinitely.
	maxOpenConns    = 25
	maxIdleConns    = 5
	connMaxLifetime = 30 * time.Minute
)
