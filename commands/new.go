package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/morph"
	"github.com/mattermost/morph/apply"
	"github.com/mattermost/morph/models"
	"github.com/spf13/cobra"

	. "github.com/dave/jennifer/jen"
)

const (
	baseDriverPath = "drivers"
	numDigits      = 6
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Bootstraps new resources",
	}

	cmd.AddCommand(
		NewDriverCmd(),
		NewScriptCmd(),
		NewPlanCmd(),
	)

	return cmd
}

func NewDriverCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "driver <name>",
		Short:         "Generates necessary file structure for a new driver",
		Args:          cobra.ExactArgs(1),
		RunE:          newDriverCmdF,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
}

func NewScriptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "script <file name>",
		Short:   "Generates new migration files",
		Example: "morph new script create_users --driver postgres --dir db/migrations --timestamp",
		Args:    cobra.ExactArgs(1),
		Run:     generateScriptCmdF,
	}

	cmd.Flags().StringP("driver", "d", "", "the driver to use.")
	cmd.Flags().BoolP("timestamp", "t", false, "a timestamp prefix will be added to migration file if set.")
	cmd.Flags().StringP("timeformat", "f", "unix", "timestamp format to be used for timestamps.")
	cmd.Flags().StringP("timezone", "z", "UTC", "time zone to be used for timestamps.")
	cmd.Flags().BoolP("sequence", "s", false, "a sequence number prefix will be added to migration file if set.")
	_ = cmd.MarkFlagRequired("driver")

	return cmd
}

func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "generate",
		Run:        generateScriptCmdF,
		Deprecated: "use `morph new script` instead.",
	}

	cmd.Flags().StringP("driver", "d", "", "the driver to use.")
	cmd.Flags().BoolP("timestamp", "t", false, "a timestamp prefix will be added to migration file if set.")
	cmd.Flags().StringP("timeformat", "f", "unix", "timestamp format to be used for timestamps.")
	cmd.Flags().StringP("timezone", "z", "UTC", "time zone to be used for timestamps.")
	cmd.Flags().BoolP("sequence", "s", false, "a sequence number prefix will be added to migration file if set.")
	_ = cmd.MarkFlagRequired("driver")

	return cmd
}

func NewPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "plan <file name>",
		Short:   "Generates new plan for the migration files with revert steps",
		Example: "morph new plan plan --driver postgres --dsn postgres://localhost:5432/morph --path db/migrations",
		Args:    cobra.ExactArgs(1),
		Run:     generatePlanCmdF,
	}

	cmd.Flags().StringP("direction", "w", "up", "the direction of the migration")
	cmd.Flags().IntP("number", "n", 0, "plan for only N migrations")
	cmd.Flags().Bool("auto", true, "generate plan with auto revert steps")

	cmd.Flags().StringP("driver", "d", "", "the database driver of the migrations")
	_ = cmd.MarkFlagRequired("driver")
	cmd.Flags().String("dsn", "", "the dsn of the database")
	_ = cmd.MarkFlagRequired("dsn")

	cmd.Flags().StringP("path", "p", "", "the source path of the migrations")
	_ = cmd.MarkFlagRequired("path")

	cmd.Flags().IntP("timeout", "t", 60, "the timeout in seconds for each migration file to run")
	cmd.Flags().StringP("migrations-table", "m", "db_migrations", "the name of the migrations table")
	cmd.Flags().StringP("lock-key", "l", "mutex_migrations", "the name of the mutex key")

	return cmd
}

func newDriverCmdF(cmd *cobra.Command, args []string) error {
	driverName := args[0]
	morph.InfoLogger.Printf("Generating necessary file structure for %q driver\n", driverName)
	driverDir := filepath.Join(baseDriverPath, driverName)

	if _, err := os.Stat(driverDir); !os.IsNotExist(err) {
		return fmt.Errorf("driver %q already exists, skipping", driverName)
	}

	return generateNewDriver(driverName)
}

func generateNewDriver(driverName string) error {
	morph.InfoLoggerLight.Printf("CreateDriver: %q: generating driver files\n", driverName)

	f := NewFile(driverName)

	f.ImportNames(map[string]string{
		"github.com/mattermost/morph/drivers": "drivers",
		"github.com/mattermost/morph/models":  "models",
	})

	f.Var().Add(Id("driverName")).Op("=").Add(Lit(driverName))
	f.Var().Add(Id("defaultConfig")).Op("=").Add(Op("&")).Id("Config").Values(Dict{
		Id("MigrationsTable"): Lit("db_migrations"),
	})
	f.Comment("add here any custom driver configuration")
	f.Var().Add(Id("configParams")).Op("=").Add(Index().String().Values())

	f.Func().Id("init").Params().Block(
		Qual("github.com/mattermost/morph/drivers", "Register").Call(Lit(driverName), Id("&"+driverName).Values()),
	)

	f.Line()
	f.Type().Id("Config").Struct(
		Id("MigrationsTable").String(),
		Comment("Add more properties here"),
	)

	f.Line()
	f.Type().Id(driverName).Struct(
		Id("config").Add(Id("*Config")),
		Comment("Add more properties here"),
	)

	f.Line()
	f.Func().Id("WithInstance").Params(Id("dbInstance").Interface(), Id("config").Add(Id("*Config"))).Params(Id("drivers.Driver"), Id("error")).BlockFunc(func(g *Group) {
		g.Return(Op("&").Id(driverName).Values(Dict{
			Id("config"): Id("config"),
		}),
			Nil(),
		)
	})

	f.Line()
	f.Comment("Implement bellow all the methods of the driver interface in order")
	f.Comment("to complete the driver functionality")

	f.Func().Params(Id("driver").Id("*"+driverName)).Id("Open").Params(Id("connURL").String()).Params(Id("drivers.Driver"), Id("error")).BlockFunc(func(g *Group) {
		g.Comment("Implement creation of the driver based on a connection URL.")
		g.Panic(Lit("implement me"))
	})

	f.Line()
	f.Func().Params(Id("driver").Id("*" + driverName)).Id("Ping").Params().Error().BlockFunc(func(g *Group) {
		g.Comment("Implement storage connection health check functionality.")
		g.Panic(Lit("implement me"))
	})

	f.Line()
	f.Func().Params(Id("driver").Id("*" + driverName)).Id("Close").Params().Error().BlockFunc(func(g *Group) {
		g.Comment("Implement functionality for tearing down the driver.")
		g.Panic(Lit("implement me"))
	})

	f.Line()
	f.Func().Params(Id("driver").Id("*" + driverName)).Id("Lock").Params().Error().BlockFunc(func(g *Group) {
		g.Comment("Implement functionality that ensures atomicity of the driver operations.")
		g.Comment("If the target storage does not need these guarantees, you can void this method.")
		g.Panic(Lit("implement me"))
	})

	f.Line()
	f.Func().Params(Id("driver").Id("*" + driverName)).Id("Unlock").Params().Error().BlockFunc(func(g *Group) {
		g.Comment("The inverse of the Lock function above.")
		g.Panic(Lit("implement me"))
	})

	f.Line()
	f.Func().Params(Id("driver").Id("*" + driverName)).Id("CreateSchemaTableIfNotExists").Params().Error().BlockFunc(func(g *Group) {
		g.Comment("Implement the creation of the migrations table onto the storage.")
		g.Panic(Lit("implement me"))
	})

	f.Line()
	f.Func().Params(Id("driver").Id("*" + driverName)).Id("Apply").Params(Id("migration").Op("*").Qual("github.com/mattermost/morph/models", "Migration")).Error().BlockFunc(func(g *Group) {
		g.Comment("Implement the functionality for apply the migration onto the storage.")
		g.Panic(Lit("implement me"))
	})

	f.Line()
	f.Func().Params(Id("driver").Id("*"+driverName)).Id("AppliedMigrations").Params().Params(Op("[]*").Qual("github.com/mattermost/morph/models", "Migration"), Error()).BlockFunc(func(g *Group) {
		g.Comment("Implement the functionality that returns which migrations has already been applied in storage.")
		g.Panic(Lit("implement me"))
	})

	driverDir := filepath.Join(baseDriverPath, driverName)
	driverFile := filepath.Join(driverDir, driverName+".go")

	morph.InfoLoggerLight.Printf("\t-- create_dir(%s)\n", driverDir)
	if err := os.Mkdir(driverDir, 0755); err != nil {
		return err
	}

	morph.InfoLoggerLight.Printf("\t-- create_file(%s)\n", driverFile)
	if err := f.Save(driverFile); err != nil {
		return err
	}

	morph.InfoLoggerLight.Printf("CreateDriver: %q: driver files generated\n", driverName)

	morph.SuccessLogger.Printf("Now you can start implementing your new driver under %q.\nThanks in advance, for contributing.\n", driverDir)

	return nil
}

func generateScriptCmdF(cmd *cobra.Command, args []string) {
	dir, _ := cmd.Flags().GetString("dir")
	driver, _ := cmd.Flags().GetString("driver")
	extension := getExtension(driver)
	fileName := args[0]

	if ts, _ := cmd.Flags().GetBool("timestamp"); ts {
		tz, _ := cmd.Flags().GetString("timezone")
		loc, err := time.LoadLocation(tz)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		date := time.Now().In(loc)

		tf, _ := cmd.Flags().GetString("timeformat")
		switch tf {
		case "unix":
			fileName = strings.Join([]string{strconv.FormatInt(date.Unix(), 10), fileName}, "_")
		case "unix-nano":
			fileName = strings.Join([]string{strconv.FormatInt(date.UnixNano(), 10), fileName}, "_")
		default:
			fileName = strings.Join([]string{date.Format(tf), fileName}, "_")
		}
	} else if seq, _ := cmd.Flags().GetBool("sequence"); seq {
		next, err := sequenceNumber(dir, extension, driver)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		version := fmt.Sprintf("%0[2]*[1]d", next, numDigits)
		if len(version) > numDigits {
			cmd.PrintErrf("next sequence number is has %d digit(s), max %d digits allowed.\n", len(version), numDigits)
			return
		}

		fileName = strings.Join([]string{version, fileName}, "_")
	}

	dir = filepath.Join(dir, driver)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		cmd.PrintErrln(err)
		return
	}

	migrations := []string{"down", "up"}
	for _, migration := range migrations {
		filePath := strings.Join([]string{filepath.Join(dir, fileName), migration, extension}, ".")
		f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		f.Close()
	}
}

func generatePlanCmdF(cmd *cobra.Command, args []string) {
	direction, _ := cmd.Flags().GetString("direction")
	direction = strings.ToLower(direction)
	limit, _ := cmd.Flags().GetInt("number")
	auto, _ := cmd.Flags().GetBool("auto")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := models.Up
	if direction == "down" {
		d = models.Down
	}

	plan, err := apply.GeneratePlan(ctx, d, limit, auto, parseEssentialFlags(cmd), parseEngineFlags(cmd)...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating plan: %s", err.Error())
		return
	}

	file, err := json.MarshalIndent(plan, "", " ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating plan: %s", err.Error())
		return
	}

	err = os.WriteFile(args[0], file, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating plan: %s", err.Error())
	}
}

func sequenceNumber(dir, extension, driver string) (int, error) {
	matches, err := filepath.Glob(filepath.Join(dir, driver, "*"+extension))
	if err != nil {
		return 0, err
	}

	if len(matches) == 0 {
		return 1, nil
	}

	filename := matches[len(matches)-1]
	matchSeqStr := filepath.Base(filename)
	idx := strings.Index(matchSeqStr, "_")

	if idx < 0 {
		return 0, fmt.Errorf("invalid migration sequence number: %s", filename)
	}

	matchSeqStr = matchSeqStr[0:idx]
	sequel, err := strconv.ParseInt(matchSeqStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return int(sequel + 1), nil
}

func getExtension(driver string) string {
	switch driver {
	case "postgres", "mysql":
		return "sql"
	default:
		return "txt"
	}
}
