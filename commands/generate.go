package commands

import (
	"github.com/spf13/cobra"
)

func GenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Creates new migrations",
		Args:  cobra.ExactArgs(1),
		Run:   generateCmdF,
	}

	cmd.Flags().StringP("driver", "d", "", "the driver to use")
	cmd.MarkFlagRequired("driver")

	return cmd
}

func generateCmdF(cmd *cobra.Command, _ []string) {
	cmd.Println("To Be Implemented")
}
