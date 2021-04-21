// +build sources
// +build !drivers

package testlib

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/go-morph/morph/models"
	"github.com/go-morph/morph/sources"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func checkMigrations(t *testing.T, migrations []*models.Migration) {
	for i := 1; i <= 3; i++ {
		migrationExists := false
		for _, migration := range migrations {
			if strings.Contains(migration.Name, fmt.Sprintf("migration_%d", i)) {
				migrationExists = true
				b, err := ioutil.ReadAll(migration.Bytes)
				require.NoError(t, err)
				assert.Contains(t, string(b), fmt.Sprintf("migration%d", i))
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
