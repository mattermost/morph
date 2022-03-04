//go:build sources && !drivers
// +build sources,!drivers

package embedded

import (
	"embed"
	"path/filepath"
	"testing"

	"github.com/mattermost/morph/sources/embedded/testdata"
	"github.com/mattermost/morph/sources/testlib"

	"github.com/stretchr/testify/require"
)

//go:embed testfiles
var assets embed.FS

func TestBindata(t *testing.T) {
	s := Resource(testdata.AssetNames(), func(name string) ([]byte, error) {
		return testdata.Asset(name)
	})

	src, err := WithInstance(s)
	require.NoError(t, err)

	testlib.Test(t, src)
}

func TestGoEmbed(t *testing.T) {
	dirEntries, err := assets.ReadDir("testfiles")
	require.NoError(t, err)

	assetNames := make([]string, len(dirEntries))
	for i, dirEntry := range dirEntries {
		assetNames[i] = dirEntry.Name()
	}

	s := Resource(assetNames, func(name string) ([]byte, error) {
		return assets.ReadFile(filepath.Join("testfiles", name))
	})

	src, err := WithInstance(s)
	require.NoError(t, err)

	testlib.Test(t, src)
}
