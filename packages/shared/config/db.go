package config

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx" driver with database/sql
)

// Connection pool bounds. Defaults are unlimited open connections, which lets
// a traffic spike exhaust Postgres' own max_connections; idle/lifetime caps
// keep the pool from holding stale connections indefinitely.
const (
	maxOpenConns    = 25
	maxIdleConns    = 5
	connMaxLifetime = 30 * time.Minute
)

// DSN returns a PostgreSQL connection string in key=value format.
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.User, c.Password, c.DBName,
	)
}

// InitDB opens a connection, verifies it with a ping, and creates all tables.
func InitDB(cfg DBConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, createTables(db)
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id           TEXT PRIMARY KEY,
			status       INTEGER NOT NULL DEFAULT 0,
			spec         JSONB NOT NULL DEFAULT '{}',
			created_at   TIMESTAMPTZ NOT NULL,
			started_at   TIMESTAMPTZ,
			finished_at  TIMESTAMPTZ,
			error_count  INTEGER NOT NULL DEFAULT 0,
			record_count INTEGER NOT NULL DEFAULT 0
		);

		CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);

		CREATE TABLE IF NOT EXISTS job_errors (
			id         BIGSERIAL PRIMARY KEY,
			job_id     TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
			record_id  TEXT NOT NULL,
			field      TEXT NOT NULL,
			message    TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_job_errors_job_id ON job_errors(job_id);

		CREATE TABLE IF NOT EXISTS aggregation_results (
			job_id      TEXT PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
			data        JSONB NOT NULL,
			computed_at TIMESTAMPTZ NOT NULL
		);
	`)
	return err
}
