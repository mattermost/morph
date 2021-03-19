package apply

import (
	"github.com/go-morph/morph"
	"github.com/go-morph/morph/sources/file"
	"github.com/spf13/cobra"
)

func Migrate(dsn string) error {
	_, err := morph.NewFromConnURL(dsn, &file.File{})
	if err != nil {
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
