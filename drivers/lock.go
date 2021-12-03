package drivers

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

const (
	// MutexTableName is the name being used for the mutex table
	MutexTableName = "db_lock"

	// MinWaitInterval is the minimum amount of time to wait between locking attempts
	MinWaitInterval = 1 * time.Second

	// MaxWaitInterval is the maximum amount of time to wait between locking attempts
	MaxWaitInterval = 5 * time.Minute

	// PollWaitInterval is the usual time to wait between unsuccessful locking attempts
	PollWaitInterval = 1 * time.Second

	// JitterWaitInterval is the amount of jitter to add when waiting to avoid thundering herds
	JitterWaitInterval = MinWaitInterval / 2

	// TTL is the interval after which a locked mutex will expire unless refreshed
	TTL = time.Second * 15

	// RefreshInterval is the interval on which the mutex will be refreshed when locked
	RefreshInterval = TTL / 2
)

// MakeLockKey returns the prefixed key used to namespace mutex keys.
func MakeLockKey(key string) (string, error) {
	if key == "" {
		return "", errors.New("must specify valid mutex key")
	}

	return key, nil
}

// NextWaitInterval determines how long to wait until the next lock retry.
func NextWaitInterval(lastWaitInterval time.Duration, err error) time.Duration {
	nextWaitInterval := lastWaitInterval

	if nextWaitInterval <= 0 {
		nextWaitInterval = MinWaitInterval
	}

	if err != nil {
		nextWaitInterval *= 2
		if nextWaitInterval > MaxWaitInterval {
			nextWaitInterval = MaxWaitInterval
		}
	} else {
		nextWaitInterval = PollWaitInterval
	}

	// Add some jitter to avoid unnecessary collision between competing other instances.
	nextWaitInterval += time.Duration(rand.Int63n(int64(JitterWaitInterval)) - int64(JitterWaitInterval)/2)

	return nextWaitInterval
}

type Mutex interface {
	Lock()
	LockWithContext(ctx context.Context) error
	Unlock()
}

type Lockable interface {
	DriverName() string
}
