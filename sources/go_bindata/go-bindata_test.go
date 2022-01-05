//go:build sources && !drivers
// +build sources,!drivers

package bindata

import (
	"testing"

	"github.com/go-morph/morph/sources/go_bindata/testdata"
	"github.com/go-morph/morph/sources/testlib"

	"github.com/stretchr/testify/require"
)

func TestBindata(t *testing.T) {
	s := Resource(testdata.AssetNames(), func(name string) ([]byte, error) {
		return testdata.Asset(name)
	})

	src, err := WithInstance(s)
	require.NoError(t, err)

	testlib.Test(t, src)
}
