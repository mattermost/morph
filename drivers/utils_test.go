package drivers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateAdvisoryLockID(t *testing.T) {
	n, err := GenerateAdvisoryLockID("mattermost", "db_migrations")
	require.NoError(t, err)

	t.Logf("advisory lock id: %s", n)
}
