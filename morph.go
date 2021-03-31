package morph

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-morph/morph/models"

	"github.com/go-morph/morph/drivers"
	"github.com/go-morph/morph/sources"

	_ "github.com/go-morph/morph/drivers/postgres"

	_ "github.com/go-morph/morph/sources/file"
)

// DefaultLockTimeout sets the max time a database driver has to acquire a lock.
var DefaultLockTimeout = 15 * time.Second

var migrationProgressStart = "==  %s: migrating ================================================="
var migrationProgressFinished = "==  %s: migrated (%s) ========================================"

type Morph struct {
	config *Config
	driver drivers.Driver
	source sources.Source
}

type Config struct {
	Logger      Logger
	LockTimeout time.Duration
}

type EngineOption func(*Morph)

var defaultConfig = &Config{
	LockTimeout: DefaultLockTimeout,
	Logger:      log.New(os.Stderr, "", log.LstdFlags), // add default logger
}

func WithLogger(logger *log.Logger) EngineOption {
	return func(m *Morph) {
		m.config.Logger = logger
	}
}

func WithLockTimeout(lockTimeout time.Duration) EngineOption {
	return func(m *Morph) {
		m.config.LockTimeout = lockTimeout
	}
}

// NewFromConnURL creates a new instance of the migrations engine from a connection url
func NewFromConnURL(connectionURL string, source sources.Source, driverName string, options ...EngineOption) (*Morph, error) {
	driver, err := drivers.Connect(connectionURL, driverName)
	if err != nil {
		return nil, err
	}

	return NewWithDriverAndSource(driver, source, options...)
}

// NewWithDriverAndSource creates a new instance of the migrations engine from an existing db instance
func NewWithDriverAndSource(driver drivers.Driver, source sources.Source, options ...EngineOption) (*Morph, error) {
	engine := &Morph{
		config: defaultConfig,
		source: source,
		driver: driver,
	}

	for _, option := range options {
		option(engine)
	}

	if err := engine.driver.CreateSchemaTableIfNotExists(); err != nil {
		return nil, err
	}

	return engine, nil
}

// ApplyAll applies all pending migrations.
func (m *Morph) ApplyAll() error {
	appliedMigrations, err := m.driver.AppliedMigrations()
	if err != nil {
		return err
	}

	pendingMigrations, err := computePendingMigrations(appliedMigrations, m.source.Migrations())
	if err != nil {
		return err
	}

	for _, migration := range pendingMigrations {
		start := time.Now()

		m.config.Logger.Printf(InfoLoggerLight.Sprintf(migrationProgressStart+"\n", migration.Name))
		if err := m.driver.Apply(migration); err != nil {
			return err
		}

		elapsed := time.Since(start)
		m.config.Logger.Printf(InfoLoggerLight.Sprintf(migrationProgressFinished+"\n", migration.Name, fmt.Sprintf("%.4fs", elapsed.Seconds())))
	}

	return nil
}

func computePendingMigrations(appliedMigrations []*models.Migration, sourceMigrations []*models.Migration) ([]*models.Migration, error) {
	// sourceMigrations has to be greater or equal to databaseMigrations
	if len(appliedMigrations) > len(sourceMigrations) {
		return nil, errors.New("migration mismatch, there are more migrations applied than those were specified in source")
	}

	dict := make(map[string]*models.Migration)
	for _, appliedMigration := range appliedMigrations {
		dict[appliedMigration.Name] = appliedMigration
	}

	var pendingMigrations []*models.Migration
	for _, sourceMigration := range sourceMigrations {
		if _, ok := dict[sourceMigration.Name]; !ok {
			pendingMigrations = append(pendingMigrations, sourceMigration)
		}
	}

	return pendingMigrations, nil
}
