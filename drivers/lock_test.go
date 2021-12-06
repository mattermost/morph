package drivers

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeLockKey(t *testing.T) {
	t.Run("fails when empty", func(t *testing.T) {
		key, err := MakeLockKey("")
		assert.Error(t, err)
		assert.Empty(t, key)
	})

	t.Run("not-empty", func(t *testing.T) {
		testCases := map[string]string{
			"key":   "key",
			"other": "other",
		}

		for key, expected := range testCases {
			actual, err := MakeLockKey(key)
			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		}
	})
}

func TestNextWaitInterval(t *testing.T) {
	t.Run("should increase the wait time", func(t *testing.T) {
		interval := time.Second
		err := errors.New("e")
		for i := 0; i < 5; i++ {
			pre := interval
			interval = NextWaitInterval(interval, err)
			require.True(t, pre < interval)
		}
	})

	t.Run("should stop increasing the wait time if reached to max", func(t *testing.T) {
		err := errors.New("e")
		interval := NextWaitInterval(maxWaitInterval, err)
		require.True(t, interval <= maxWaitInterval)
	})

	t.Run("should return default interval if no error", func(t *testing.T) {
		interval := NextWaitInterval(maxWaitInterval, nil)
		require.True(t, interval <= pollWaitInterval)
	})
}
