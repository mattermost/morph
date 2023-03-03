package commands

import (
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "morph",
		Short:   "A database migration tool",
		Version: "v1.0.4",
	}

	cmd.PersistentFlags().String("dir", ".", "the migrations directory")

	cmd.AddCommand(
		ApplyCmd(),
		NewCmd(),
		NewGenerateCmd(),
	)

	return cmd
}
