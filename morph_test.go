//go:build !drivers
// +build !drivers

package morph

import (
	"context"
	"errors"
	"testing"

	"github.com/mattermost/morph/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestSortMigrations(t *testing.T) {
	testCases := []struct {
		Name          string
		Migrations    []string
		ExpectedOrder []string
	}{
		{
			Name:          "sequence based migration names",
			Migrations:    []string{"000002_migration", "000003_migration", "000001_migration"},
			ExpectedOrder: []string{"000001_migration", "000002_migration", "000003_migration"},
		},
		{
			Name:          "timestamp based migration names",
			Migrations:    []string{"202103221430_migration_3", "202103221400_migration_2", "202103221321_migration_1"},
			ExpectedOrder: []string{"202103221321_migration_1", "202103221400_migration_2", "202103221430_migration_3"},
		},
	}

	migrationsFromNames := func(names []string) []*models.Migration {
		migrations := []*models.Migration{}
		for _, name := range names {
			migrations = append(migrations, &models.Migration{RawName: name})
		}
		return migrations
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			sortedMigrations := sortMigrations(migrationsFromNames(tc.Migrations))

			for i, migration := range sortedMigrations {
				assert.Equalf(t, tc.ExpectedOrder[i], migration.RawName, "Expected migration %q to be in position %d, but found %q instead", tc.ExpectedOrder[i], i, migration.RawName)
			}
		})
	}
}

func TestApplyAll(t *testing.T) {
	h := newTestHelper(t)
	defer h.Teardown(t)

	h.AddMigration(t, "test_migration")

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		err := engine.ApplyAll()
		require.NoError(t, err)

		migrations, err := engine.driver.AppliedMigrations()
		require.NoError(t, err)

		require.Len(t, migrations, 1)
	})
}

func TestApply(t *testing.T) {
	h := newTestHelper(t)
	defer h.Teardown(t)

	h.AddMigration(t, "test_migration_2")

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		_, err := engine.Apply(1)
		require.NoError(t, err)

		migrations, err := engine.driver.AppliedMigrations()
		require.NoError(t, err)

		require.Len(t, migrations, 1)
	})
}

func TestDiff(t *testing.T) {
	h := newTestHelper(t)
	defer h.Teardown(t)

	h.AddMigration(t, "test_migration_3")
	h.AddMigration(t, "test_migration_4")

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		// we have 2 pending migrations in the source
		// the diff should return 2 migrations upwards
		migrations, err := engine.Diff(models.Up)
		require.NoError(t, err)

		require.Len(t, migrations, 2)
	}, "should have 2 migrations to apply")

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		// There should be no migrations downwards
		// since we didn't apply any migrations yet
		migrations, err := engine.Diff(models.Down)
		require.NoError(t, err)

		require.Empty(t, migrations)
	}, "should return an empty list for down migrations")

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		// We apply the first migration, so we should have 1 migration pending
		_, err := engine.Apply(1)
		require.NoError(t, err)

		migrations, err := engine.Diff(models.Up)
		require.NoError(t, err)

		require.Len(t, migrations, 1)
	}, "there should only one migration to apply")

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		// Apply all remaining migrations, so we should have no migrations pending
		err := engine.ApplyAll()
		require.NoError(t, err)

		migrations, err := engine.Diff(models.Up)
		require.NoError(t, err)

		require.Empty(t, migrations)
	}, "there should be no migrations to apply")

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		// We can now have 2 migrations to rollback
		migrations, err := engine.Diff(models.Down)
		require.NoError(t, err)

		require.Len(t, migrations, 2)
	}, "should have 2 migrations to downgrade")
}

func TestOppositeMigrations(t *testing.T) {
	h := newTestHelper(t).CreateBasicMigrations(t)
	defer h.Teardown(t)

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		// Check for applied migrations first, so we can test the opposite
		// Should return empty since there are no applied migrations
		migrations, err := engine.driver.AppliedMigrations()
		require.NoError(t, err)

		migrations, err = engine.GetOppositeMigrations(migrations)
		require.NoError(t, err)

		require.Empty(t, migrations)
	}, "no migrations applied empty list should be returned")

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		// Apply one pending migration, should have one migration to rollback
		_, err := engine.Apply(1)
		require.NoError(t, err)

		migrations, err := engine.driver.AppliedMigrations()
		require.NoError(t, err)

		rollbackMigrations, err := engine.GetOppositeMigrations(migrations)
		require.NoError(t, err)

		require.Len(t, migrations, 1)
		require.Equal(t, models.Down, rollbackMigrations[0].Direction)
		require.Equal(t, migrations[0].Name, rollbackMigrations[0].Name)
	}, "one migration applied, reverse migration should be returned")

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		migrations := []*models.Migration{
			{Name: "202103221321_migration_1", Direction: models.Up},
			{Name: "202103221400_migration_2", Direction: models.Down},
		}
		rollbackMigrations, err := engine.GetOppositeMigrations(migrations)
		require.EqualError(t, err, "migrations have different directions")
		require.Empty(t, rollbackMigrations)
	}, "error when migrations have different directions")

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		err := engine.ApplyAll()
		require.NoError(t, err)

		migrations, err := engine.driver.AppliedMigrations()
		require.NoError(t, err)

		rollbackMigrations, err := engine.GetOppositeMigrations(migrations)
		require.NoError(t, err)

		require.Len(t, migrations, 3)
		for i := range rollbackMigrations {
			require.Equal(t, models.Down, rollbackMigrations[i].Direction)
			require.Equal(t, migrations[i].Name, rollbackMigrations[i].Name)
		}
	})
}

func TestGeneratePlan(t *testing.T) {
	h := newTestHelper(t).CreateBasicMigrations(t)
	defer h.Teardown(t)

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		migrations, err := engine.Diff(models.Up)
		require.NoError(t, err)

		plan, err := engine.GeneratePlan(migrations, true)
		require.NoError(t, err)

		require.ElementsMatch(t, plan.Migrations, migrations)
		require.Len(t, plan.RevertMigrations, len(migrations))
	})
}

func TestApplyPlan(t *testing.T) {
	h := newTestHelper(t).CreateBasicMigrations(t)
	defer h.Teardown(t)

	h.RunForAllDrivers(t, func(t *testing.T, engine *Morph) {
		migrations, err := engine.Diff(models.Up)
		require.NoError(t, err)

		plan, err := engine.GeneratePlan(migrations, true)
		require.NoError(t, err)

		err = engine.ApplyPlan(plan)
		require.NoError(t, err)

	})

	td := &testDriver{failAt: 3, mode: models.Up}
	ts := &basicSource{
		migrations: []*models.Migration{
			{Name: "000001_migration_a", Direction: models.Up, Version: 1, RawName: "000001_migration_a.up.sql"},
			{Name: "000002_migration_b", Direction: models.Up, Version: 2, RawName: "000002_migration_b.up.sql"},
			{Name: "000003_migration_c", Direction: models.Up, Version: 3, RawName: "000003_migration_c.up.sql"},
			{Name: "000004_migration_d", Direction: models.Up, Version: 4, RawName: "000004_migration_d.up.sql"},
			{Name: "000001_migration_a", Direction: models.Down, Version: 1, RawName: "000001_migration_a.down.sql"},
			{Name: "000002_migration_b", Direction: models.Down, Version: 2, RawName: "000002_migration_b.down.sql"},
			{Name: "000003_migration_c", Direction: models.Down, Version: 3, RawName: "000003_migration_c.down.sql"},
			{Name: "000004_migration_d", Direction: models.Down, Version: 4, RawName: "000004_migration_d.down.sql"},
		},
	}

	t.Run("should rollback when error occurs/mockDriver", func(t *testing.T) {
		engine, err := New(context.Background(), td, ts)
		require.NoError(t, err)

		migrations, err := engine.Diff(models.Up)
		require.NoError(t, err)

		require.Len(t, migrations, 4)
		plan, err := engine.GeneratePlan(migrations, true)
		require.NoError(t, err)

		err = engine.ApplyPlan(plan)
		require.EqualError(t, err, "could not apply migration: failed to apply migration")

		migrations, err = engine.Diff(models.Up)
		require.NoError(t, err)
		require.Len(t, migrations, 4)
	})

	t.Run("should not rollback all if no error/mockDriver", func(t *testing.T) {
		td.failAt = 0 // don't fail
		engine, err := New(context.Background(), td, ts)
		require.NoError(t, err)

		migrations, err := engine.Diff(models.Up)
		require.NoError(t, err)

		require.Len(t, migrations, 4)
		plan, err := engine.GeneratePlan(migrations, true)
		require.NoError(t, err)

		err = engine.ApplyPlan(plan)
		require.NoError(t, err)

		migrations, err = engine.Diff(models.Up)
		require.NoError(t, err)
		require.Len(t, migrations, 0)

		migrations, err = engine.Diff(models.Down)
		require.NoError(t, err)
		require.Len(t, migrations, 4)
	})

	t.Run("there are already applied migrations, should only revert applied/mockDriver", func(t *testing.T) {
		td.failAt = 4 // fail at version 3
		td.applied = []*models.Migration{ts.migrations[0], ts.migrations[1]}

		engine, err := New(context.Background(), td, ts)
		require.NoError(t, err)

		migrations, err := engine.Diff(models.Up)
		require.NoError(t, err)

		require.Len(t, migrations, 2)
		plan, err := engine.GeneratePlan(migrations, true)
		require.NoError(t, err)

		err = engine.ApplyPlan(plan)
		require.EqualError(t, err, "could not apply migration: failed to apply migration")

		migrations, err = engine.Diff(models.Up)
		require.NoError(t, err)
		require.Len(t, migrations, 2)

		migrations, err = engine.Diff(models.Down)
		require.NoError(t, err)
		require.Len(t, migrations, 2)
	})

	t.Run("we aim to rollback and rollback fails/mockDriver", func(t *testing.T) {
		td.failAt = 2 // fail at version 3
		td.mode = models.Down
		td.applied = []*models.Migration{ts.migrations[0], ts.migrations[1], ts.migrations[2]}

		engine, err := New(context.Background(), td, ts)
		require.NoError(t, err)

		migrations, err := engine.Diff(models.Down)
		require.NoError(t, err)
		require.Len(t, migrations, 3)

		plan, err := engine.GeneratePlan(migrations, true)
		require.NoError(t, err)

		err = engine.ApplyPlan(plan)
		require.EqualError(t, err, "could not apply migration: failed to apply migration")

		migrations, err = engine.Diff(models.Down)
		require.NoError(t, err)
		require.Len(t, migrations, 3)

		migrations, err = engine.Diff(models.Up)
		require.NoError(t, err)
		require.Len(t, migrations, 1)
	})
}

type basicSource struct {
	migrations []*models.Migration
}

func (s *basicSource) Migrations() []*models.Migration {
	return s.migrations
}

type testDriver struct {
	failAt  int
	applied []*models.Migration
	mode    models.Direction
}

func (d *testDriver) Ping() error {
	return nil
}

func (d *testDriver) Close() error {
	return nil
}

// Apply applies a migration and imitates what a real driver would do. The trick is
// it's going to throw an error if the migration version is equal to the failAt value.
func (d *testDriver) Apply(migration *models.Migration, saveVersion bool) error {
	if int(migration.Version) == d.failAt && d.mode == migration.Direction {
		return errors.New("failed to apply migration")
	}

	if !saveVersion {
		return nil
	}

	if migration.Direction == models.Down {
		for i := range d.applied {
			if d.applied[i].Name == migration.Name {
				d.applied = append(d.applied[:i], d.applied[i+1:]...)
				break
			}
		}
	} else {
		d.applied = append(d.applied, migration)
	}

	return nil
}

func (d *testDriver) AppliedMigrations() ([]*models.Migration, error) {
	return d.applied, nil
}

func (d *testDriver) SetConfig(key string, value interface{}) error {
	return nil
}
