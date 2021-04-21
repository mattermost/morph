// +build sources
// +build !drivers

package bindata

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/go-morph/morph/models"
	"github.com/go-morph/morph/sources/go_bindata/testdata"

	"github.com/stretchr/testify/require"
)

func TestBindata(t *testing.T) {
	checkMigration := func(t *testing.T, migrations []*models.Migration, i int) {
		migration := migrations[i-1]
		require.Contains(t, migration.Name, fmt.Sprintf("migration_%d", i))
		b, err := ioutil.ReadAll(migration.Bytes)
		require.NoError(t, err)
		require.Contains(t, string(b), fmt.Sprintf("migration%d", i))
	}

	s := Resource(testdata.AssetNames(), func(name string) ([]byte, error) {
		return testdata.Asset(name)
	})

	src, err := WithInstance(s)
	require.NoError(t, err)

	migrations := src.Migrations()
	require.Len(t, migrations, 3)

	checkMigration(t, migrations, 1)
	checkMigration(t, migrations, 2)
	checkMigration(t, migrations, 3)
}
