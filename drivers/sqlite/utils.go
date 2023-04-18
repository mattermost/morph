package sqlite

import "github.com/mattermost/morph/drivers"

func getDefaultConfig() *driverConfig {
	return &driverConfig{
		Config: drivers.Config{
			MigrationsTable:        "db_migrations",
			StatementTimeoutInSecs: 300,
			MigrationMaxSize:       defaultMigrationMaxSize,
		},
	}
}
