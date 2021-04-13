// +build sources
// +build !drivers

package file

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-morph/morph/models"

	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	testFilesDir := "../../testfiles"

	checkMigration := func(t *testing.T, migrations []*models.Migration, i int) {
		migration := migrations[i-1]
		require.Contains(t, migration.FileName, fmt.Sprintf("migration_%d", i))
		b, err := ioutil.ReadAll(migration.Bytes)
		require.NoError(t, err)
		require.Contains(t, string(b), fmt.Sprintf("migration%d", i))
	}

	t.Run("should correctly create a source with the testfiles", func(t *testing.T) {
		sourceURL := "file://" + testFilesDir
		f, err := (&File{}).Open(sourceURL)
		require.NoError(t, err)

		migrations := f.Migrations()
		require.Len(t, migrations, 3)

		checkMigration(t, migrations, 1)
		checkMigration(t, migrations, 2)
		checkMigration(t, migrations, 3)
	})

	t.Run("should work correctly as well if the path is absolute", func(t *testing.T) {
		absTestFilesDir, err := filepath.Abs(testFilesDir)
		require.NoError(t, err)

		f, err := (&File{}).Open(absTestFilesDir)
		require.NoError(t, err)

		migrations := f.Migrations()
		require.Len(t, migrations, 3)

		checkMigration(t, migrations, 1)
		checkMigration(t, migrations, 2)
		checkMigration(t, migrations, 3)
	})
}
