package drivers

import (
	"fmt"
	"log"
	"sync"

	"github.com/go-morph/morph/models"
)

var driversMu sync.RWMutex
var registeredDrivers = make(map[string]Driver)

type Driver interface {
	Open(connURL string) (Driver, error)
	Ping() error
	CreateSchemaTableIfNotExists() error
	Close() error
	Lock() error
	Unlock() error
	Apply(migration *models.Migration) error
	AppliedMigrations() ([]*models.Migration, error)
	Logger() log.Logger
}

func Connect(connectionURL, driverName string) (Driver, error) {
	driversMu.RLock()
	driver, ok := registeredDrivers[driverName]
	driversMu.RUnlock()

	if !ok {
		return nil, &AppError{
			OrigErr: nil,
			Driver:  driverName,
			Message: "unsupported driver found",
		}
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

func GetDriver(driverName string) (Driver, error) {
	driversMu.Lock()
	defer driversMu.Unlock()

	driver, ok := registeredDrivers[driverName]
	if !ok {
		return nil, fmt.Errorf("driver %s not found", driverName)
	}

	return driver, nil
}
