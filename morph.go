package morph

import (
	"log"
	"time"

	"github.com/go-morph/morph/drivers"
	"github.com/go-morph/morph/sources"

	_ "github.com/go-morph/morph/drivers/postgres"

	_ "github.com/go-morph/morph/sources/file"
)

// DefaultLockTimeout sets the max time a database driver has to acquire a lock.
var DefaultLockTimeout = 15 * time.Second

type Morph struct {
	config *Config
	driver drivers.Driver
	source sources.Source
}

type Config struct {
	Logger      *log.Logger
	LockTimeout time.Duration
}

type MorphOption func(*Morph)

var defaultConfig = &Config{
	LockTimeout: DefaultLockTimeout,
	Logger:      &log.Logger{}, // add default logger
}

func WithLogger(logger *log.Logger) MorphOption {
	return func(m *Morph) {
		m.config.Logger = logger
	}
}

func WithLockTimeout(lockTimeout time.Duration) MorphOption {
	return func(m *Morph) {
		m.config.LockTimeout = lockTimeout
	}
}

// NewFromConnURL creates a new instance of the migrations engine from a connection url
func NewFromConnURL(connectionURL string, source sources.Source, driverName string, options ...MorphOption) (*Morph, error) {
	driver, err := drivers.Connect(connectionURL, driverName)
	if err != nil {
		return nil, err
	}

	return NewWithDriverAndSource(driver, source, options...)
}

// NewWithDriverAndSource creates a new instance of the migrations engine from an existing db instance
func NewWithDriverAndSource(driver drivers.Driver, source sources.Source, options ...MorphOption) (*Morph, error) {
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

	InfoLoggerLight.Printf("* Found %d applied migrations in the database.\n", len(appliedMigrations))
	InfoLoggerLight.Printf("* Found %d migrations to be applied from source.\n", len(m.source.Migrations()))
	return nil
}
