package main

import "time"

const (
	// Server timeouts.
	serverReadTimeout  = 15 * time.Second
	serverWriteTimeout = 60 * time.Second
	shutdownTimeout    = 15 * time.Second

	// Default environment variable values.
	defaultPort   = "8080"
	defaultDBHost = "localhost"
	defaultDBPort = "5432"
	defaultDBUser = "postgres"
	defaultDBName = "pipeline_db"
)
