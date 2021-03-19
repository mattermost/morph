package drivers

import "log"

type config struct {
	logger *log.Logger
	migrationsTableName string
}

const defaultMigrationsTableName = "schema_migrations"
var defaultLogger = &log.Logger{}

func NewConfig(logger *log.Logger, migrationsTableName string) *config {
	cfg := &config{
		logger: logger,
		migrationsTableName: migrationsTableName,
	}

	if logger == nil {
		cfg.logger = defaultLogger
	}

	if migrationsTableName == "" {
		cfg.migrationsTableName = defaultMigrationsTableName
	}

	return cfg
}

