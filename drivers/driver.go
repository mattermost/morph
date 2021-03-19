package drivers

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/go-morph/morph/models"
	"log"
	"net/url"
	"sync"
)

var driversMu sync.RWMutex
var registeredDrivers = make(map[string]Driver)

type Driver interface {
	Open(connURL string) (Driver, error)
	Ping() error
	CreateSchemaTable() error
	Close() error
	Lock() error
	UnLock() error
	Apply(migration *models.Migration) error
	AppliedMigrations() ([]*models.Migration, error)
	Logger() log.Logger
}

func Connect(connectionURL string) (Driver, error) {
	uri, err := url.Parse(connectionURL)
	if err != nil {
		return nil, errors.Wrap(err, "unsupported scheme found: ")
	}
	driversMu.RLock()
	driver, ok := registeredDrivers[uri.Scheme]
	driversMu.RUnlock()

	if !ok {
		return nil, errors.New(fmt.Sprintf("unsupported driver %s found", uri.Scheme))
	}

	connectedDriver, err := driver.Open(connectionURL)
	if err != nil {
		return nil, err
	}

	if err := connectedDriver.Ping(); err != nil {
		return nil, err
	}

	return connectedDriver, nil
}

func Register(driverName string, driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	registeredDrivers[driverName] = driver
}