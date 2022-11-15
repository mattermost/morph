package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattermost/morph"
	"github.com/mattermost/morph/apply"
	"github.com/mattermost/morph/models"
	"github.com/spf13/cobra"
)

func ApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Applies migrations",
	}

	// Required flags
	cmd.PersistentFlags().StringP("driver", "d", "", "the database driver of the migrations")
	_ = cmd.MarkPersistentFlagRequired("driver")
	cmd.PersistentFlags().String("dsn", "", "the dsn of the database")
	_ = cmd.MarkPersistentFlagRequired("dsn")
	cmd.PersistentFlags().StringP("path", "p", "", "the source path of the migrations")
	_ = cmd.MarkPersistentFlagRequired("path")

	// Optional flags
	cmd.PersistentFlags().IntP("timeout", "t", 60, "the timeout in seconds for each migration file to run")
	cmd.PersistentFlags().StringP("migrations-table", "m", "db_migrations", "the name of the migrations table")
	cmd.PersistentFlags().StringP("lock-key", "l", "mutex_migrations", "the name of the mutex key")
	cmd.PersistentFlags().Bool("dry-run", false, "prints the plan without applying it")

	// Add subcommands
	cmd.AddCommand(
		UpApplyCmd(),
		DownApplyCmd(),
		MigrateApplyCmd(),
		PlanApplyCmd(),
	)

	return cmd
}

func UpApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "up",
		Short:         "Apply migrations forward a number of steps",
		RunE:          upApplyCmdF,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	cmd.Flags().Int("number", 0, "apply N up migrations")
	return cmd
}

func DownApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "down",
		Short:         "Apply migrations backwards a number of steps",
		RunE:          downApplyCmdF,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.Flags().Int("number", 0, "apply N down migrations")
	return cmd
}

func MigrateApplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "migrate",
		Short:         "Apply all migrations",
		RunE:          migrateApplyCmdF,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
}

func PlanApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "plan <file name>",
		Short:         "Apply the plan",
		RunE:          planApplyCmdF,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	cmd.Flags().String("plan", "plan.morph", "apply plan")

	return cmd
}

func upApplyCmdF(cmd *cobra.Command, _ []string) error {
	steps, _ := cmd.Flags().GetInt("number")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	morph.InfoLogger.Printf("Attempting to apply %d migrations...\n", steps)
	n, err := apply.Up(ctx, steps, parseEssentialFlags(cmd), parseEngineFlags(cmd)...)
	if n > 0 {
		morph.SuccessLogger.Printf("%d migrations applied.\n", n)
	} else if n == 0 {
		morph.InfoLogger.Println("no migrations applied.")
	}
	return err
}

func downApplyCmdF(cmd *cobra.Command, _ []string) error {
	steps, _ := cmd.Flags().GetInt("number")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	morph.InfoLogger.Printf("Attempting to apply  %d migrations...\n", steps)
	n, err := apply.Down(ctx, steps, parseEssentialFlags(cmd), parseEngineFlags(cmd)...)
	if n > 0 {
		morph.SuccessLogger.Printf("%d migrations applied.\n", n)
	} else if n == 0 {
		morph.InfoLogger.Println("no migrations applied.")
	}
	return err
}

func migrateApplyCmdF(cmd *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	morph.InfoLogger.Println("Applying all pending migrations...")
	if err := apply.Migrate(ctx, parseEssentialFlags(cmd), parseEngineFlags(cmd)...); err != nil {
		return err
	}
	morph.SuccessLogger.Println("Pending migrations applied.")

	return nil
}

func planApplyCmdF(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := os.Open(args[0])
	if err != nil {
		return err
	}

	var plan models.Plan
	err = json.NewDecoder(f).Decode(&plan)
	if err != nil {
		return err
	}

	morph.InfoLogger.Printf("Attempting to apply plan...\n")
	err = apply.Plan(ctx, &plan, parseEssentialFlags(cmd), parseEngineFlags(cmd)...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error applying plan: %s", err.Error())
		return err
	}
	morph.InfoLogger.Printf("Successfully applied the plan.\n")

	return nil
}

// parseEssentialFlags parses the essential flags for the apply command.
// which are the DSN, the driver and the source path.
func parseEssentialFlags(cmd *cobra.Command) apply.ConnectionParameters {
	dsn, _ := cmd.Flags().GetString("dsn")
	driverName, _ := cmd.Flags().GetString("driver")
	path, _ := cmd.Flags().GetString("path")

	return apply.ConnectionParameters{
		DSN:        dsn,
		DriverName: driverName,
		SourcePath: path,
	}
}

// parseEngineFlags parses the optional engine flags for the apply commands.
func parseEngineFlags(cmd *cobra.Command) []morph.EngineOption {
	timeout, _ := cmd.Flags().GetInt("timeout")
	tableName, _ := cmd.Flags().GetString("migrations-table")
	mutexKey, _ := cmd.Flags().GetString("lock-key")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	return []morph.EngineOption{
		morph.SetMigrationTableName(tableName),
		morph.SetStatementTimeoutInSeconds(timeout),
		morph.WithLock(mutexKey),
		morph.SetDryRun(dryRun),
	}
}
