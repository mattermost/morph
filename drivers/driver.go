package drivers

import (
	"github.com/go-morph/morph/models"
)

type Driver interface {
	Ping() error
	Close() error
	Lock() error
	Unlock() error
	Apply(migration *models.Migration, saveVersion bool) error
	AppliedMigrations() ([]*models.Migration, error)
	SetConfig(key string, value interface{}) error
}
