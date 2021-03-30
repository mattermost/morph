package apply

import (
	"github.com/go-morph/morph"
	"github.com/go-morph/morph/sources"
	"github.com/spf13/cobra"
)

func Migrate(dsn, source, driverName, path string) error {
	src, err := sources.Open(source, path)
	if err != nil {
		return err
	}
	defer src.Close()

	engine, err := morph.NewFromConnURL(dsn, src, driverName)
	if err != nil {
		return err
	}

	return engine.ApplyAll()
}

func Up(arge cobra.PositionalArgs) error {
	return nil
}

func Down(arge cobra.PositionalArgs) error {
	return nil
}
