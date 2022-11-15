package morph

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mattermost/morph/models"

	"github.com/mattermost/morph/drivers"
	"github.com/mattermost/morph/sources"

	ms "github.com/mattermost/morph/drivers/mysql"
	ps "github.com/mattermost/morph/drivers/postgres"

	_ "github.com/mattermost/morph/sources/embedded"
	_ "github.com/mattermost/morph/sources/file"
)

var migrationProgressStart = "==  %s: migrating  ================================================="
var migrationProgressFinished = "==  %s: migrated (%s)  ========================================"

const maxProgressLogLength = 100

type Morph struct {
	config *Config
	driver drivers.Driver
	source sources.Source
	mutex  drivers.Locker
}

type Config struct {
	Logger  Logger
	LockKey string
}

type EngineOption func(*Morph) error

func WithLogger(logger Logger) EngineOption {
	return func(m *Morph) error {
		m.config.Logger = logger
		return nil
	}
}

func SetMigrationTableName(name string) EngineOption {
	return func(m *Morph) error {
		return m.driver.SetConfig("MigrationsTable", name)
	}
}

func SetStatementTimeoutInSeconds(n int) EngineOption {
	return func(m *Morph) error {
		return m.driver.SetConfig("StatementTimeoutInSecs", n)
	}
}

// WithLock creates a lock table in the database so that the migrations are
// guaranteed to be executed from a single instance. The key is used for naming
// the mutex.
func WithLock(key string) EngineOption {
	return func(m *Morph) error {
		m.config.LockKey = key
		return nil
	}
}

// New creates a new instance of the migrations engine from an existing db instance and a migrations source.
// If the driver implements the Lockable interface, it will also wait until it has acquired a lock.
// The context is propagated to the drivers lock method (if the driver implements divers.Locker interface) and
// it can be used to cancel the lock acquisition.
func New(ctx context.Context, driver drivers.Driver, source sources.Source, options ...EngineOption) (*Morph, error) {
	engine := &Morph{
		config: &Config{
			Logger: newColorLogger(log.New(os.Stderr, "", log.LstdFlags)), // add default logger
		},
		source: source,
		driver: driver,
	}

	for _, option := range options {
		if err := option(engine); err != nil {
			return nil, fmt.Errorf("could not apply option: %w", err)
		}
	}

	if err := driver.Ping(); err != nil {
		return nil, err
	}

	if impl, ok := driver.(drivers.Lockable); ok && engine.config.LockKey != "" {
		var mx drivers.Locker
		var err error
		switch impl.DriverName() {
		case "mysql":
			mx, err = ms.NewMutex(engine.config.LockKey, driver, engine.config.Logger)
		case "postgres":
			mx, err = ps.NewMutex(engine.config.LockKey, driver, engine.config.Logger)
		default:
			err = errors.New("driver does not support locking")
		}
		if err != nil {
			return nil, err
		}

		engine.mutex = mx
		err = mx.Lock(ctx)
		if err != nil {
			return nil, err
		}
	}

	return engine, nil
}

// Close closes the underlying database connection of the engine.
func (m *Morph) Close() error {
	if m.mutex != nil {
		err := m.mutex.Unlock()
		if err != nil {
			return err
		}
	}

	return m.driver.Close()
}

func (m *Morph) apply(migration *models.Migration, saveVersion bool) error {
	start := time.Now()
	migrationName := migration.Name
	m.config.Logger.Println(formatProgress(fmt.Sprintf(migrationProgressStart, migrationName)))
	if err := m.driver.Apply(migration, saveVersion); err != nil {
		return err
	}

	elapsed := time.Since(start)
	m.config.Logger.Println(formatProgress(fmt.Sprintf(migrationProgressFinished, migrationName, fmt.Sprintf("%.4fs", elapsed.Seconds()))))

	return nil
}

// ApplyAll applies all pending migrations.
func (m *Morph) ApplyAll() error {
	_, err := m.Apply(-1)
	return err
}

// Applies limited number of migrations upwards.
func (m *Morph) Apply(limit int) (int, error) {
	appliedMigrations, err := m.driver.AppliedMigrations()
	if err != nil {
		return -1, err
	}

	pendingMigrations, err := computePendingMigrations(appliedMigrations, m.source.Migrations())
	if err != nil {
		return -1, err
	}

	migrations := make([]*models.Migration, 0)
	sortedMigrations := sortMigrations(pendingMigrations)

	for _, migration := range sortedMigrations {
		if migration.Direction != models.Up {
			continue
		}
		migrations = append(migrations, migration)
	}

	steps := limit
	if len(migrations) < steps {
		return -1, fmt.Errorf("there are only %d migrations available, but you requested %d", len(migrations), steps)
	}

	if limit < 0 {
		steps = len(migrations)
	}

	var applied int
	for i := 0; i < steps; i++ {
		if err := m.apply(migrations[i], true); err != nil {
			return applied, err
		}
		applied++
	}

	return applied, nil
}

// ApplyDown rollbacks a limited number of migrations
// if limit is given below zero, all down scripts are going to be applied.
func (m *Morph) ApplyDown(limit int) (int, error) {
	appliedMigrations, err := m.driver.AppliedMigrations()
	if err != nil {
		return -1, err
	}

	sortedMigrations := reverseSortMigrations(appliedMigrations)
	downMigrations, err := findDownScripts(sortedMigrations, m.source.Migrations())
	if err != nil {
		return -1, err
	}

	steps := limit
	if len(sortedMigrations) < steps {
		return -1, fmt.Errorf("there are only %d migrations available, but you requested %d", len(sortedMigrations), steps)
	}

	if limit < 0 {
		steps = len(sortedMigrations)
	}

	var applied int
	for i := 0; i < steps; i++ {
		migrationName := sortedMigrations[i].Name
		if err := m.apply(downMigrations[migrationName], true); err != nil {
			return applied, err
		}
		applied++
	}

	return applied, nil
}

// Diff returns the difference between the applied migrations and the available migrations.
func (m *Morph) Diff(mode models.Direction) ([]*models.Migration, error) {
	appliedMigrations, err := m.driver.AppliedMigrations()
	if err != nil {
		return nil, err
	}

	if mode == models.Down {
		sortedMigrations := reverseSortMigrations(appliedMigrations)
		downMigrations, err := findDownScripts(sortedMigrations, m.source.Migrations())
		if err != nil {
			return nil, err
		}

		diff := make([]*models.Migration, 0, len(downMigrations))
		for i := 0; i < len(sortedMigrations); i++ {
			diff = append(diff, downMigrations[sortedMigrations[i].Name])
		}

		return diff, nil
	}

	pendingMigrations, err := computePendingMigrations(appliedMigrations, m.source.Migrations())
	if err != nil {
		return nil, err
	}

	var diff []*models.Migration
	for _, migration := range sortMigrations(pendingMigrations) {
		if migration.Direction != models.Up {
			continue
		}
		diff = append(diff, migration)
	}

	return diff, nil
}

func (m *Morph) GetOppositeMigrations(migrations []*models.Migration) ([]*models.Migration, error) {
	var direction models.Direction
	migrationsMap := make(map[string]*models.Migration)
	for _, migration := range migrations {
		if direction == "" {
			direction = migration.Direction
		}
		// check if the migrations has the same direction
		if direction != migration.Direction {
			return nil, errors.New("migrations have different directions")
		}

		migrationsMap[migration.Name] = migration
	}

	rollbackMigrations := make([]*models.Migration, 0, len(migrations))
	availableMigrations := m.source.Migrations()
	for _, migration := range availableMigrations {
		// skip if we have the same direction for the migration
		// we are looking for opposite direction
		if migration.Direction == direction {
			continue
		}

		// we don't have the migration in the map
		// so we can't rollback it
		_, ok := migrationsMap[migration.Name]
		if !ok {
			continue
		}

		rollbackMigrations = append(rollbackMigrations, migration)
	}

	if len(migrations) != len(rollbackMigrations) {
		return nil, errors.New("not all migrations have opposite migrations")
	}

	return rollbackMigrations, nil
}

// GeneratePlan returns the plan to apply these migrations and also includes
// the safe rollback steps for the given migrations.
func (m *Morph) GeneratePlan(migrations []*models.Migration) (*models.Plan, error) {
	rollbackMigrations, err := m.GetOppositeMigrations(migrations)
	if err != nil {
		return nil, fmt.Errorf("could not get opposite migrations: %w", err)
	}

	plan := models.NewPlan(migrations, rollbackMigrations)

	return plan, nil
}

func (m *Morph) ApplyPlan(plan *models.Plan) error {
	if err := plan.Validate(); err != nil {
		return fmt.Errorf("invalid plan: %w", err)
	}

	revertMigrations := make([]*models.Migration, 0, len(plan.RevertMigrations))
	var err error
	var failIndex int

	for i := range plan.Migrations {
		// add to the revert queue
		for _, migration := range plan.RevertMigrations {
			if migration.Name == plan.Migrations[i].Name && migration.Version == plan.Migrations[i].Version {
				revertMigrations = append(revertMigrations, migration)
				break
			}
		}

		err = m.apply(plan.Migrations[i], true)
		if err != nil {
			break
		}

		failIndex = i
	}

	if err == nil {
		return nil
	}

	m.config.Logger.Printf("migration %s failed, starting rollback", plan.Migrations[failIndex].Name)

	for j := len(revertMigrations) - 1; j >= 0; j-- {
		// There is a special case when we are reverting a rollback
		// We shouldn't save the version if we are trying to restore the last applied migration
		// here is an example, lets say we have following migrations in the applied migrations table:
		// migration_1, migration_2, migration_3
		// Once we initiate the rollback, we will have the following:
		// migration_3, migration_2, migration_1 (to rollback)
		// Let's say we have a bug in migration_2 and failed.
		// We don't remove that version from the database, because migration is not successfully rolled back.
		// So in this case, we need to apply the migration_2 (up) but it will be in the migrations table.
		// Therefore we are not saving the version in the database because it will fail on the save version step.
		skipSave := revertMigrations[j].Direction == models.Up && j == len(revertMigrations)-1
		rErr := m.apply(revertMigrations[j], !skipSave)
		if rErr != nil {
			return fmt.Errorf("could not rollback migrations after trying to migrate: %w", rErr)
		}

		m.config.Logger.Printf("successfully rolled back migration: %s", revertMigrations[j].Name)
	}

	// return error in any case
	return fmt.Errorf("could not apply migration: %w", err)
}

func reverseSortMigrations(migrations []*models.Migration) []*models.Migration {
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version > migrations[j].Version
	})
	return migrations
}

func sortMigrations(migrations []*models.Migration) []*models.Migration {
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].RawName < migrations[j].RawName
	})
	return migrations
}

func computePendingMigrations(appliedMigrations []*models.Migration, sourceMigrations []*models.Migration) ([]*models.Migration, error) {
	// sourceMigrations has to be greater or equal to databaseMigrations
	if len(appliedMigrations) > len(sourceMigrations) {
		return nil, errors.New("migration mismatch, there are more migrations applied than those were specified in source")
	}

	dict := make(map[string]*models.Migration)
	for _, appliedMigration := range appliedMigrations {
		dict[appliedMigration.Name] = appliedMigration
	}

	var pendingMigrations []*models.Migration
	for _, sourceMigration := range sourceMigrations {
		if _, ok := dict[sourceMigration.Name]; !ok {
			pendingMigrations = append(pendingMigrations, sourceMigration)
		}
	}

	return pendingMigrations, nil
}

func findDownScripts(appliedMigrations []*models.Migration, sourceMigrations []*models.Migration) (map[string]*models.Migration, error) {
	tmp := make(map[string]*models.Migration)
	for _, m := range sourceMigrations {
		if m.Direction != models.Down {
			continue
		}
		tmp[m.Name] = m
	}

	for _, m := range appliedMigrations {
		_, ok := tmp[m.Name]
		if !ok {
			return nil, fmt.Errorf("could not find down script for %s", m.Name)
		}
	}

	return tmp, nil
}

func formatProgress(p string) string {
	if len(p) < maxProgressLogLength {
		return p + strings.Repeat("=", maxProgressLogLength-len(p))
	}

	if len(p) > maxProgressLogLength {
		return p[:maxProgressLogLength]
	}

	return p
}
