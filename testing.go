package morph

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/mgdelacroix/foundation"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/morph/drivers"
	"github.com/mattermost/morph/drivers/mysql"
	"github.com/mattermost/morph/models"
	"github.com/mattermost/morph/sources/embedded"
)

type TestHelper struct {
	*foundation.Foundation
}

func NewTestHelper(t *testing.T, e *Morph) (*TestHelper, func()) {
	th := &TestHelper{
		Foundation: foundation.New(t, NewMigrator(e)),
	}

	return th, th.TearDown
}

// Migrator is a utility class for testing migrations. It can be used in Foundation testing
// see https://github.com/mgdelacroix/foundation
type Migrator struct {
	db         *sql.DB
	driverName string
	engine     *Morph
}

// New creates a new instance of the Migrator to test migrations.
func NewMigrator(engine *Morph) *Migrator {
	dn, ok := engine.driver.(drivers.DBNamer)
	if !ok {
		panic("driver does not implement DBNamer")
	}

	return &Migrator{
		db:         dn.DB(),
		driverName: dn.Name(),
		engine:     engine,
	}
}

func (m *Migrator) DB() *sql.DB {
	return m.db
}

func (m *Migrator) DriverName() string {
	return m.driverName
}

func (m *Migrator) Setup() error {
	return nil
}

func (m *Migrator) MigrateToStep(step int) error {
	migrations, err := m.engine.driver.AppliedMigrations()
	if err != nil {
		return err
	}

	if len(migrations) > step {
		return fmt.Errorf("asked to migrate to step %d, but there are already %d migrations applied", step, len(migrations))
	}

	_, err = m.engine.Apply(step - len(migrations))
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %s", err)
	}

	return nil
}

func (m *Migrator) Interceptors() map[int]func() error {
	// switch direction {
	// case models.Up:
	return m.engine.intercecptorsUp
	// case models.Down:
	// 	return m.engine.intercecptorsDown
	// }
	// return nil
}

func (m *Migrator) TearDown() error {
	return m.engine.Close()
}

func TestSomething(t *testing.T) {
	// part 1: initialize the test helper
	db, err := sql.Open("mysql", "<DSN>")
	require.NoError(t, err)

	driver, err := mysql.WithInstance(db)
	require.NoError(t, err)

	src, err := embedded.WithInstance(migrationAssets)
	require.NoError(t, err)

	engine, err := New(context.Background(), driver, src, opts...)
	require.NoError(t, err)

	engine.AddInterceptor(12, models.Up, func() error {
		// do something
		return nil
	})

	th, teardown := NewTestHelper(t, engine)
	defer teardown()

	// part 2: do actual testing
	th.MigrateToStep(12)
	th.ExecFile("my_query.sql")
	th.DB().Get(&struct{}{}, "")
}
