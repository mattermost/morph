package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattermost/morph"
	"github.com/spf13/cobra"

	. "github.com/dave/jennifer/jen"
)

const baseDriverPath = "drivers"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Bootstraps new resources",
	}

	cmd.AddCommand(
		NewDriverCmd(),
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
	morph.InfoLoggerLight.Printf("== CreateDriver: %q: generating driver files\n", driverName)
	morph.InfoLoggerLight.Println("=================================================")

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

	morph.InfoLoggerLight.Printf("== CreateDriver: %q: driver files generated\n", driverName)
	morph.InfoLoggerLight.Println("=================================================")

	morph.SuccessLogger.Printf("Now you can start implementing your new driver under %q.\nThanks in advance, for contributing.\n", driverDir)

	return nil
}
