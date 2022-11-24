package apply

import (
	"context"
	"fmt"

	"github.com/mattermost/morph"
	"github.com/mattermost/morph/drivers"
	"github.com/mattermost/morph/drivers/mysql"
	"github.com/mattermost/morph/drivers/postgres"
	"github.com/mattermost/morph/drivers/sqlite"
	"github.com/mattermost/morph/models"
	"github.com/mattermost/morph/sources/file"
)

type ConnectionParameters struct {
	DSN        string
	DriverName string
	SourcePath string
}

func Migrate(ctx context.Context, params ConnectionParameters, options ...morph.EngineOption) error {
	engine, err := initializeEngine(ctx, params.DSN, params.DriverName, params.SourcePath, options...)
	if err != nil {
		return err
	}
	defer engine.Close()

	return engine.ApplyAll()
}

func Up(ctx context.Context, limit int, params ConnectionParameters, options ...morph.EngineOption) (int, error) {
	engine, err := initializeEngine(ctx, params.DSN, params.DriverName, params.SourcePath, options...)
	if err != nil {
		return -1, err
	}
	defer engine.Close()

	return engine.Apply(limit)
}

func Down(ctx context.Context, limit int, params ConnectionParameters, options ...morph.EngineOption) (int, error) {
	engine, err := initializeEngine(ctx, params.DSN, params.DriverName, params.SourcePath, options...)
	if err != nil {
		return -1, err
	}
	defer engine.Close()

	return engine.ApplyDown(limit)
}

func Plan(ctx context.Context, plan *models.Plan, params ConnectionParameters, options ...morph.EngineOption) error {
	engine, err := initializeEngine(ctx, params.DSN, params.DriverName, params.SourcePath, options...)
	if err != nil {
		return err
	}
	defer engine.Close()

	return engine.ApplyPlan(plan)
}

func GeneratePlan(ctx context.Context, direction models.Direction, limit int, auto bool, params ConnectionParameters, options ...morph.EngineOption) (*models.Plan, error) {
	engine, err := initializeEngine(ctx, params.DSN, params.DriverName, params.SourcePath, options...)
	if err != nil {
		return nil, err
	}
	defer engine.Close()

	migrations, err := engine.Diff(direction)
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(migrations) > limit {
		migrations = migrations[:limit]
	}

	return engine.GeneratePlan(migrations, auto)
}

func initializeEngine(ctx context.Context, dsn, driverName, path string, options ...morph.EngineOption) (*morph.Morph, error) {
	src, err := file.Open(path)
	if err != nil {
		return nil, err
	}

	var driver drivers.Driver
	switch driverName {
	case "mysql":
		driver, err = mysql.Open(dsn)
	case "postgresql", "postgres":
		driver, err = postgres.Open(dsn)
	case "sqlite":
		driver, err = sqlite.Open(dsn)
	default:
		err = fmt.Errorf("unsupported driver %s", driverName)
	}
	if err != nil {
		return nil, err
	}

	engine, err := morph.New(ctx, driver, src, options...)
	if err != nil {
		return nil, err
	}

	return engine, err
}
