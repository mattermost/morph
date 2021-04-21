// +build sources
// +build !drivers

package file

import (
	"path/filepath"
	"testing"

	"github.com/go-morph/morph/sources/testlib"

	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	testFilesDir := "../../testfiles"

	t.Run("should correctly create a source with the testfiles", func(t *testing.T) {
		sourceURL := "file://" + testFilesDir
		f, err := (&File{}).Open(sourceURL)
		require.NoError(t, err)

		testlib.Test(t, f)
	})

	t.Run("should work correctly as well if the path is absolute", func(t *testing.T) {
		absTestFilesDir, err := filepath.Abs(testFilesDir)
		require.NoError(t, err)

		f, err := (&File{}).Open(absTestFilesDir)
		require.NoError(t, err)

		testlib.Test(t, f)
	})
}
