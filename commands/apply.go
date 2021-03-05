package commands

import (
	"github.com/spf13/cobra"
)

func ApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Applies migrations",
	}

	cmd.AddCommand(
		UpApplyCmd(),
		DownApplyCmd(),
		MigrateApplyCmd(),
	)

	return cmd
}

func UpApplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Apply migrations forward a number of steps",
		Run:   upApplyCmdF,
	}
}

func DownApplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Apply migrations backwards a number of steps",
		Run:   downApplyCmdF,
	}
}

func MigrateApplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Apply all migrations",
		Run:   migrateApplyCmdF,
	}
}

func upApplyCmdF(cmd *cobra.Command, _ []string) {
	cmd.Println("To Be Implemented")
}

func downApplyCmdF(cmd *cobra.Command, _ []string) {
	cmd.Println("To Be Implemented")
}

func migrateApplyCmdF(cmd *cobra.Command, _ []string) {
	cmd.Println("To Be Implemented")
}
