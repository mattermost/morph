package bindata

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/go-morph/morph/models"
	"github.com/go-morph/morph/sources"
)

type AssetFunc func(name string) ([]byte, error)

func Resource(names []string, fn AssetFunc) *AssetSource {
	return &AssetSource{
		Names:     names,
		AssetFunc: fn,
	}
}

type AssetSource struct {
	Names     []string
	AssetFunc AssetFunc
}

func init() {
	sources.Register("go-bindata", &Bindata{})
}

type Bindata struct {
	path        string
	assetSource *AssetSource
	migrations  []*models.Migration
}

func (b *Bindata) Open(url string) (sources.Source, error) {
	return nil, fmt.Errorf("not implemented")
}

func WithInstance(assetSource *AssetSource) (sources.Source, error) {
	b := &Bindata{
		path:        "<embed::go-bindata>",
		assetSource: assetSource,
		migrations:  []*models.Migration{},
	}

	for _, filename := range assetSource.Names {
		migrationBytes, err := b.assetSource.AssetFunc(filename)
		if err != nil {
			return nil, fmt.Errorf("cannot read migration %q: %w", filename, err)
		}

		m := &models.Migration{
			FileName: filename,
			Bytes:    ioutil.NopCloser(bytes.NewReader(migrationBytes)),
		}

		b.migrations = append(b.migrations, m)
	}

	return b, nil
}

func (b *Bindata) Close() error {
	return nil
}

func (b *Bindata) Migrations() []*models.Migration {
	return b.migrations
}
