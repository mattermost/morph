package apply

import (
	"github.com/go-morph/morph"
	"github.com/go-morph/morph/sources"
	"github.com/spf13/cobra"
)

func Migrate(dsn string, source string) error {
	src, err := sources.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	if _, err := morph.NewFromConnURL(dsn, src); err != nil {
		return err
	}

	return nil
}

func Up(arge cobra.PositionalArgs) error {
	return nil
}

func Down(arge cobra.PositionalArgs) error {
	return nil
}
