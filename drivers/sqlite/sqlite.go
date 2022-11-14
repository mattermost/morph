package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/mattermost/morph/drivers"
	"github.com/mattermost/morph/models"
	_ "modernc.org/sqlite"
)

const driverName = "sqlite"
const defaultMigrationMaxSize = 10 * 1 << 20 // 10 MB

// add here any custom driver configuration
var configParams = []string{
	"x-migration-max-size",
	"x-migrations-table",
	"x-statement-timeout",
}

type driverConfig struct {
	drivers.Config
	closeDBonClose bool
}

type sqlite struct {
	conn   *sql.Conn
	db     *sql.DB
	config *driverConfig

	lockedFlag int32 // indicates that the driver is locked or not
}

func WithInstance(dbInstance *sql.DB) (drivers.Driver, error) {
	conn, err := dbInstance.Conn(context.Background())
	if err != nil {
		return nil, &drivers.DatabaseError{Driver: driverName, Command: "grabbing_connection", OrigErr: err, Message: "failed to grab connection to the database"}
	}

	return &sqlite{config: getDefaultConfig(), conn: conn, db: dbInstance}, nil
}

func Open(filePath string) (drivers.Driver, error) {
	customParams, err := drivers.ExtractCustomParams(filePath, configParams)
	if err != nil {
		return nil, &drivers.AppError{Driver: driverName, OrigErr: err, Message: "failed to parse custom parameters from url"}
	}

	sanitizedConnURL, err := drivers.RemoveParamsFromURL(filePath, configParams)
	if err != nil {
		return nil, &drivers.AppError{Driver: driverName, OrigErr: err, Message: "failed to sanitize url from custom parameters"}
	}

	sanitizedConnURL = strings.TrimSuffix(sanitizedConnURL, "?")

	driverConfig, err := mergeConfigWithParams(customParams, getDefaultConfig())
	if err != nil {
		return nil, &drivers.AppError{Driver: driverName, OrigErr: err, Message: "failed to merge custom params to driver config"}
	}

	if _, err := os.Stat(sanitizedConnURL); errors.Is(err, os.ErrNotExist) {
		return nil, &drivers.AppError{Driver: driverName, OrigErr: err, Message: "failed to open db file"}
	}

	db, err := sql.Open(driverName, sanitizedConnURL)
	if err != nil {
		return nil, &drivers.DatabaseError{Driver: driverName, Command: "opening_connection", OrigErr: err, Message: "failed to open connection with the database"}
	}

	conn, err := db.Conn(context.Background())
	if err != nil {
		return nil, &drivers.DatabaseError{Driver: driverName, Command: "grabbing_connection", OrigErr: err, Message: "failed to grab connection to the database"}
	}

	driverConfig.closeDBonClose = true

	return &sqlite{
		conn:   conn,
		db:     db,
		config: driverConfig,
	}, nil
}

func (driver *sqlite) Ping() error {
	ctx, cancel := drivers.GetContext(driver.config.StatementTimeoutInSecs)
	defer cancel()

	return driver.conn.PingContext(ctx)
}

func (sqlite) DriverName() string {
	return driverName
}

func (driver *sqlite) Close() error {
	if driver.conn != nil {
		if err := driver.conn.Close(); err != nil {
			return &drivers.DatabaseError{
				OrigErr: err,
				Driver:  driverName,
				Message: "failed to close database connection",
				Command: "sqlite_conn_close",
				Query:   nil,
			}
		}
	}

	if driver.db != nil && driver.config.closeDBonClose {
		if err := driver.db.Close(); err != nil {
			return &drivers.DatabaseError{
				OrigErr: err,
				Driver:  driverName,
				Message: "failed to close database",
				Command: "sqlite_db_close",
				Query:   nil,
			}
		}
		driver.db = nil
	}

	driver.conn = nil
	return nil
}

func (driver *sqlite) lock() error {
	if !atomic.CompareAndSwapInt32(&driver.lockedFlag, 0, 1) {
		return &drivers.DatabaseError{
			OrigErr: errors.New("already locked"),
			Driver:  driverName,
			Message: "failed to obtain lock",
			Command: "lock_driver",
		}
	}

	return nil
}

func (driver *sqlite) unlock() error {
	atomic.StoreInt32(&driver.lockedFlag, 0)

	return nil
}

func (driver *sqlite) createSchemaTableIfNotExists() (err error) {
	ctx, cancel := drivers.GetContext(driver.config.StatementTimeoutInSecs)
	defer cancel()

	createTableIfNotExistsQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (Version bigint not null primary key, Name varchar not null)", driver.config.MigrationsTable)
	if _, err = driver.conn.ExecContext(ctx, createTableIfNotExistsQuery); err != nil {
		return &drivers.DatabaseError{
			OrigErr: err,
			Driver:  driverName,
			Message: "failed while executing query",
			Command: "create_migrations_table_if_not_exists",
			Query:   []byte(createTableIfNotExistsQuery),
		}
	}

	return nil
}

func (driver *sqlite) Apply(migration *models.Migration, saveVersion bool) (err error) {
	if err = driver.lock(); err != nil {
		return err
	}
	defer func() {
		_ = driver.unlock()
	}()

	query := migration.Query()

	ctx, cancel := drivers.GetContext(driver.config.StatementTimeoutInSecs)
	defer cancel()

	transaction, err := driver.conn.BeginTx(ctx, nil)
	if err != nil {
		return &drivers.DatabaseError{
			OrigErr: err,
			Driver:  driverName,
			Message: "error while opening a transaction to the database",
			Command: "begin_transaction",
		}
	}

	if err = execTransaction(transaction, query); err != nil {
		return err
	}

	if saveVersion {
		updateVersionQuery := driver.addMigrationQuery(migration)
		if err = execTransaction(transaction, updateVersionQuery); err != nil {
			return err
		}
	}

	err = transaction.Commit()
	if err != nil {
		return &drivers.DatabaseError{
			OrigErr: err,
			Driver:  driverName,
			Message: "error while committing a transaction to the database",
			Command: "commit_transaction",
		}
	}

	return nil
}

func (driver *sqlite) AppliedMigrations() (migrations []*models.Migration, err error) {
	if driver.conn == nil {
		return nil, &drivers.AppError{
			OrigErr: errors.New("driver has no connection established"),
			Message: "database connection is missing",
			Driver:  driverName,
		}
	}

	if err = driver.lock(); err != nil {
		return nil, err
	}
	defer func() {
		_ = driver.unlock()
	}()

	if err := driver.createSchemaTableIfNotExists(); err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT version, name FROM %s", driver.config.MigrationsTable)
	ctx, cancel := drivers.GetContext(driver.config.StatementTimeoutInSecs)
	defer cancel()
	var appliedMigrations []*models.Migration
	var version uint32
	var name string

	rows, err := driver.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, &drivers.DatabaseError{
			OrigErr: err,
			Driver:  driverName,
			Message: "failed to fetch applied migrations",
			Command: "select_applied_migrations",
			Query:   []byte(query),
		}
	}
	defer rows.Close()

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
			Name:      name,
			Version:   version,
			Direction: models.Up,
		})
	}

	return appliedMigrations, nil
}

func mergeConfigWithParams(params map[string]string, config *driverConfig) (*driverConfig, error) {
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

func (driver *sqlite) addMigrationQuery(migration *models.Migration) string {
	if migration.Direction == models.Down {
		return fmt.Sprintf("DELETE FROM %s WHERE (Version=%d AND NAME='%s')", driver.config.MigrationsTable, migration.Version, migration.Name)
	}
	return fmt.Sprintf("INSERT INTO %s (Version, Name) VALUES (%d, '%s')", driver.config.MigrationsTable, migration.Version, migration.Name)
}

func (driver *sqlite) SetConfig(key string, value interface{}) error {
	if driver.config != nil {
		switch key {
		case "StatementTimeoutInSecs":
			n, ok := value.(int)
			if ok {
				driver.config.StatementTimeoutInSecs = n
				return nil
			}
			return fmt.Errorf("incorrect value type for %s", key)
		case "MigrationsTable":
			n, ok := value.(string)
			if ok {
				driver.config.MigrationsTable = n
				return nil
			}
			return fmt.Errorf("incorrect value type for %s", key)
		}
	}

	return fmt.Errorf("incorrect key name %q", key)
}

func execTransaction(transaction *sql.Tx, query string) error {
	if _, err := transaction.Exec(query); err != nil {
		if txErr := transaction.Rollback(); txErr != nil {
			err = errors.Wrap(errors.New(err.Error()+txErr.Error()), "failed to execute query in migration transaction")

			return &drivers.DatabaseError{
				OrigErr: err,
				Driver:  driverName,
				Command: "rollback_transaction",
			}
		}

		return &drivers.DatabaseError{
			OrigErr: err,
			Driver:  driverName,
			Message: "failed when applying migration",
			Command: "apply_migration",
			Query:   []byte(query),
		}
	}

	return nil
}
