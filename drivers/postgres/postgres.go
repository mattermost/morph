package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/go-morph/morph/drivers"
	"github.com/go-morph/morph/models"
	_ "github.com/lib/pq"
)

var (
	driverName    = "postgres"
	defaultConfig = &Config{
		MigrationsTable:        "schema_migrations",
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
	drivers.Register("postgres", &postgres{})
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

	conn, err := db.Conn(context.Background())
	if err != nil {
		return nil, &drivers.DatabaseError{Driver: driverName, Command: "grabbing_connection", OrigErr: err, Message: "failed to grab connection to the database"}
	}

	if driverConfig.DatabaseName, err = extractDatabaseName(connURL); err != nil {
		return nil, &drivers.AppError{Driver: driverName, OrigErr: err, Message: "failed to extract database name from connection url"}
	}

	if driverConfig.SchemaName, err = currentSchema(conn, driverConfig); err != nil {
		return nil, err
	}

	pg.db = db
	pg.config = driverConfig
	pg.conn = conn

	return pg, nil
}

func currentSchema(conn *sql.Conn, config *Config) (string, error) {
	query := "SELECT CURRENT_SCHEMA()"

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.StatementTimeoutInSecs)*time.Second)
	defer cancel()

	var schemaName string
	if err := conn.QueryRowContext(ctx, query).Scan(&schemaName); err != nil {
		return "", &drivers.DatabaseError{
			OrigErr: err,
			Driver:  driverName,
			Message: "failed to fetch current schema",
			Command: "current_schema",
			Query:   []byte(query),
		}
	}
	return schemaName, nil
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(pg.config.StatementTimeoutInSecs)*time.Second)
	defer cancel()

	return pg.conn.PingContext(ctx)
}

func (pg *postgres) CreateSchemaTableIfNotExists() (err error) {
	if pg.conn == nil {
		return &drivers.AppError{
			OrigErr: errors.New("driver has no connection established"),
			Message: "database connection is missing",
			Driver:  driverName,
		}
	}

	if err = pg.Lock(); err != nil {
		return err
	}
	defer func() {
		// If we saw no error prior to unlocking and unlocking returns an error we need to
		// assign the unlocking error to err
		if unlockErr := pg.Unlock(); unlockErr != nil && err == nil {
			err = unlockErr
		}
	}()

	var count int
	userHasCreatePermissionsQuery := fmt.Sprintf("SELECT COUNT(1) FROM information_schema.tables WHERE table_name = '%s' AND table_schema = (SELECT current_schema()) LIMIT 1", pg.config.MigrationsTable)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(pg.config.StatementTimeoutInSecs)*time.Second)
	defer cancel()

	row := pg.conn.QueryRowContext(ctx, userHasCreatePermissionsQuery)
	if err = row.Scan(&count); err != nil {
		return &drivers.DatabaseError{
			OrigErr: err,
			Driver:  driverName,
			Message: "failed while executing query",
			Command: "check_if_migrations_table_exists",
			Query:   []byte(userHasCreatePermissionsQuery),
		}
	}

	if count == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(pg.config.StatementTimeoutInSecs)*time.Second)
		defer cancel()

		createTableIfNotExistsQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (version bigint not null primary key, name varchar not null)", pg.config.MigrationsTable)
		if _, err = pg.conn.ExecContext(ctx, createTableIfNotExistsQuery); err != nil {
			return &drivers.DatabaseError{
				OrigErr: err,
				Driver:  driverName,
				Message: "failed while executing query",
				Command: "create_migrations_table_if_not_exists",
				Query:   []byte(createTableIfNotExistsQuery),
			}
		}
	}

	return nil
}

func (pg *postgres) Close() error {
	if pg.conn != nil {
		if err := pg.conn.Close(); err != nil {
			return &drivers.DatabaseError{
				OrigErr: err,
				Driver:  driverName,
				Message: "failed to close database connection",
				Command: "pg_conn_close",
				Query:   nil,
			}
		}
	}

	if pg.db != nil {
		if err := pg.db.Close(); err != nil {
			return &drivers.DatabaseError{
				OrigErr: err,
				Driver:  driverName,
				Message: "failed to close database connection",
				Command: "pg_conn_close",
				Query:   nil,
			}
		}
	}

	return nil
}

func (pg *postgres) Lock() error {
	aid, err := drivers.GenerateAdvisoryLockID(pg.config.DatabaseName, pg.config.SchemaName)
	if err != nil {
		return err
	}

	// This will wait until the lock can be acquired or until the statement timeout has reached.
	query := "SELECT pg_advisory_lock($1)"
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(pg.config.StatementTimeoutInSecs)*time.Second)
	defer cancel()

	if _, err := pg.conn.ExecContext(ctx, query, aid); err != nil {
		return &drivers.DatabaseError{
			OrigErr: err,
			Driver:  driverName,
			Message: "failed to obtain advisory lock",
			Command: "lock_migrations_table",
			Query:   []byte(query),
		}
	}

	return nil
}

func (pg *postgres) Unlock() error {
	aid, err := drivers.GenerateAdvisoryLockID(pg.config.DatabaseName, pg.config.SchemaName)
	if err != nil {
		return err
	}

	query := "SELECT pg_advisory_unlock($1)"
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(pg.config.StatementTimeoutInSecs)*time.Second)
	defer cancel()

	if _, err := pg.conn.ExecContext(ctx, query, aid); err != nil {
		return &drivers.DatabaseError{
			OrigErr: err,
			Driver:  driverName,
			Message: "failed to unlock advisory lock",
			Command: "unlock_migrations_table",
			Query:   []byte(query),
		}
	}

	return nil
}

func (pg *postgres) Apply(migration *models.Migration) error {
	panic("implement me")
}

func (pg *postgres) AppliedMigrations() (migrations []*models.Migration, err error) {
	if err = pg.Lock(); err != nil {
		return nil, err
	}
	defer func() {
		// If we saw no error prior to unlocking and unlocking returns an error we need to
		// assign the unlocking error to err
		if unlockErr := pg.Unlock(); unlockErr != nil && err == nil {
			err = unlockErr
		}
	}()

	query := fmt.Sprintf("SELECT version, name FROM %s", pg.config.MigrationsTable)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(pg.config.StatementTimeoutInSecs)*time.Second)
	defer cancel()
	var appliedMigrations []*models.Migration
	var version uint32
	var name string

	rows, err := pg.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, &drivers.DatabaseError{
			OrigErr: err,
			Driver:  driverName,
			Message: "failed to fetch applied migrations",
			Command: "select_applied_migrations",
			Query:   []byte(query),
		}
	}

	for rows.Next() {
		if err := rows.Scan(&version, &name); err != nil {
			return nil, &drivers.DatabaseError{
				OrigErr: err,
				Driver:  driverName,
				Message: "failed to scan applied migration row",
				Command: "scan_applied_migrations",
			}
		}

		appliedMigrations = append(appliedMigrations, &models.Migration{
			Name:    name,
			Version: version,
		})
	}

	return appliedMigrations, nil
}

func (pg *postgres) Logger() log.Logger {
	panic("implement me")
}
