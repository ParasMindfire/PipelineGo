package config

// DBConfig holds PostgreSQL connection details, typically loaded from environment variables.
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}
