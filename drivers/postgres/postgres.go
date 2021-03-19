package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/go-morph/morph/drivers"
	"github.com/go-morph/morph/models"
	_ "github.com/lib/pq"
)

var (
	driverName    = "postgres"
	defaultConfig = &Config{
		MigrationsTable:        "schema_migrations",
		DatabaseName:           "",
		SchemaName:             "",
		StatementTimeoutInSecs: 5,
		MigrationMaxSize:       defaultMigrationMaxSize,
	}
	defaultMigrationMaxSize = 10 * 1 << 20 // 10 MB
	configParams            = []string{
		"x-migration-max-size",
		"x-migrations-table",
		"x-statement-timeout",
	}
)

func init() {
	db := postgres{}
	drivers.Register("postgres", &db)
	drivers.Register("postgresql", &db)
}

type Config struct {
	MigrationsTable        string
	DatabaseName           string
	SchemaName             string
	StatementTimeoutInSecs int
	MigrationMaxSize       int
}

type postgres struct {
	conn   *sql.Conn
	db     *sql.DB
	config *Config
}

func (pg *postgres) Open(connURL string) (drivers.Driver, error) {
	customParams, err := drivers.ParseCustomParams(connURL, configParams)
	if err != nil {
		return nil, &drivers.AppError{Driver: driverName, OrigErr: err, Message: "failed to parse custom parameters from url"}
	}

	sanitizedConnURL, err := drivers.SanitizeConnURL(connURL, configParams)
	if err != nil {
		return nil, &drivers.AppError{Driver: driverName, OrigErr: err, Message: "failed to sanitize url from custom parameters"}
	}

	driverConfig, err := mergeConfigWithParams(customParams, defaultConfig)
	if err != nil {
		return nil, &drivers.AppError{Driver: driverName, OrigErr: err, Message: "failed to merge custom params to driver config"}
	}

	db, err := sql.Open(driverName, sanitizedConnURL)
	if err != nil {
		return nil, &drivers.DatabaseError{Driver: driverName, Command: "opening_connection", OrigErr: err, Message: "failed to open connection with the database"}
	}

	postgres := &postgres{
		db:     db,
		config: driverConfig,
	}

	return postgres, nil
}

func mergeConfigWithParams(params map[string]string, config *Config) (*Config, error) {
	var err error

	for _, configKey := range configParams {
		if v, ok := params[configKey]; ok {
			switch configKey {
			case "x-migration-max-size":
				if config.MigrationMaxSize, err = strconv.Atoi(v); err != nil {
					return nil, errors.New(fmt.Sprintf("failed to cast config param %s of %s", configKey, v))
				}
			case "x-migrations-table":
				config.MigrationsTable = v
			case "x-statement-timeout":
				if config.StatementTimeoutInSecs, err = strconv.Atoi(v); err != nil {
					return nil, errors.New(fmt.Sprintf("failed to cast config param %s of %s", configKey, v))
				}
			}
		}
	}

	return config, nil
}

func (pg *postgres) Ping() error {
	return pg.db.Ping()
}

func (pg *postgres) CreateSchemaTable() error {
	panic("implement me")
}

func (pg *postgres) Close() error {
	panic("implement me")
}

func (pg *postgres) Lock() error {
	panic("implement me")
}

func (pg *postgres) UnLock() error {
	panic("implement me")
}

func (pg *postgres) Apply(migration *models.Migration) error {
	panic("implement me")
}

func (pg *postgres) AppliedMigrations() ([]*models.Migration, error) {
	panic("implement me")
}

func (pg *postgres) Logger() log.Logger {
	panic("implement me")
}
