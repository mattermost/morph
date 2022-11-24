package commands

import (
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "morph",
		Short:   "A database migration tool",
		Version: "v0.1",
	}

	cmd.PersistentFlags().String("dir", ".", "the migrations directory")

	cmd.AddCommand(
		ApplyCmd(),
		NewCmd(),
		NewGenerateCmd(),
	)

	return cmd
}
