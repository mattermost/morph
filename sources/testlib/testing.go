//go:build sources && !drivers
// +build sources,!drivers

package testlib

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mattermost/morph/models"
	"github.com/mattermost/morph/sources"

	"github.com/stretchr/testify/assert"
)

func checkMigrations(t *testing.T, migrations []*models.Migration) {
	for i := 1; i <= 3; i++ {
		migrationExists := false
		for _, migration := range migrations {
			if strings.Contains(migration.Name, fmt.Sprintf("migration_%d", i)) {
				migrationExists = true
				assert.Contains(t, string(migration.Bytes), fmt.Sprintf("migration%d", i))
			}
		}
		assert.Truef(t, migrationExists, "Migration %d was not found in source", i)
	}
}

func Test(t *testing.T, src sources.Source) {
	migrations := src.Migrations()
	assert.Len(t, migrations, 3)
	checkMigrations(t, migrations)
}
