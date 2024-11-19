package drivers

import (
	"github.com/mattermost/morph/models"
)

type Config struct {
	// MigrationsTableName is the name of the table that will store the migrations.
	MigrationsTable string
	// StatementTimeoutInSecs is used to set a timeout for each migration file.
	// Set below zero to disable timeout. Zero value will result in default value, which is 60 seconds.
	StatementTimeoutInSecs int
	// MigrationMaxSize is the maximum size of a migration file in bytes.
	MigrationMaxSize int
}

// Driver is the interface that should be implemented by all drivers.
// The driver is responsible for applying migrations to the database.
type Driver interface {
	Ping() error
	// Close closes the underlying db connection. If the driver is created via Open() function
	// this method will also going to call Close() on the sql.db instance.
	Close() error
	// Apply should apply the migration to the database. If saveVersion is true, the driver should
	// save the migration version in the database.
	Apply(migration *models.Migration, saveVersion bool) error
	// AppliedMigrations should return a list of applied migrations.
	// Ideally migrations should be sorted by version.
	AppliedMigrations() ([]*models.Migration, error)
	// SetConfig should be used to set the driver configuration. The key is the name of the configuration
	// This method should return an error if the key is not supported.
	// This method is being used by the morph engine to apply configurations such as:
	// StatementTimeoutInSecs
	// MigrationsTableName
	SetConfig(key string, value interface{}) error
}
