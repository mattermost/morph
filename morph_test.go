package morph

import (
	"testing"

	"github.com/mattermost/morph/models"

	"github.com/stretchr/testify/assert"
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
