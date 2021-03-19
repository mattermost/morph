package commands

import (
	"github.com/go-morph/morph"
	"github.com/go-morph/morph/apply"
	"github.com/spf13/cobra"
)

var dsn string

func ApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Applies migrations",
	}

	cmd.PersistentFlags().StringVarP(&dsn, "dsn", "d", "", "the dsn of the database")
	cmd.MarkPersistentFlagRequired("dsn")

	cmd.AddCommand(
		UpApplyCmd(),
		DownApplyCmd(),
		MigrateApplyCmd(),
	)

	return cmd
}

func UpApplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "up",
		Short:         "Apply migrations forward a number of steps",
		RunE:          upApplyCmdF,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
}

func DownApplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "down",
		Short:         "Apply migrations backwards a number of steps",
		RunE:          downApplyCmdF,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
}

func MigrateApplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "migrate",
		Short:         "Apply all migrations",
		RunE:          migrateApplyCmdF,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
}

func upApplyCmdF(cmd *cobra.Command, _ []string) error {
	return apply.Up(cmd.Args)
}

func downApplyCmdF(cmd *cobra.Command, _ []string) error {
	return apply.Down(cmd.Args)
}

func migrateApplyCmdF(cmd *cobra.Command, _ []string) error {
	morph.InfoLogger.Println("Applying all pending migrations...")
	if err := apply.Migrate(dsn); err != nil {
		return err
	}
	morph.SuccessLogger.Println("Pending migrations applied.")

	return nil
}
