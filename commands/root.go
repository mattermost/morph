package commands

import (
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "morph",
		Short: "A database migration tool",
	}

	cmd.PersistentFlags().String("dir", ".", "the migrations directory")

	cmd.AddCommand(
		ApplyCmd(),
		GenerateCmd(),
		NewCmd(),
	)

	return cmd
}
