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

// Creates a new instance of the migrations engine
func NewFromConnURL(connectionURL string, source sources.Source, options ...MorphOption) (*Morph, error) {
	driver, err := drivers.Connect(connectionURL)
	if err != nil {
		return nil, err
	}

	return NewWithDriverAndSource(driver, source, options...), nil
}

// Creates a new instance of the migrations engine from an existing db instance
func NewWithDriverAndSource(driver drivers.Driver, source sources.Source, options ...MorphOption) *Morph {
	engine := &Morph{
		config: defaultConfig,
		source: source,
		driver: driver,
	}

	for _, option := range options {
		option(engine)
	}

	return engine
}
