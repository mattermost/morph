package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mattermost/morph/drivers"
	"github.com/mattermost/morph/models"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	testConnURL    = "morph-test.db"
	defaultConnURL = "morph-default.db"
)

type SqliteTestSuite struct {
	suite.Suite
}

func (suite *SqliteTestSuite) InitializeDriver(connURL string) drivers.Driver {
	connectedDriver, err := Open(connURL)
	suite.Require().NoError(err, "should not error when connecting to database from url")
	suite.Require().NotNil(connectedDriver)

	return connectedDriver
}

func (suite *SqliteTestSuite) TestOpen() {
	suite.T().Run("when connURL is valid and bare(no custom configuration present)", func(t *testing.T) {
		connectedDriver := suite.InitializeDriver(testConnURL)
		t.Cleanup(func() {
			require.NoError(t, connectedDriver.Close(), "should close the driver w/o errors")
		})
	})

	suite.T().Run("when connURL is invalid", func(t *testing.T) {
		_, err := Open("something invalid")

		suite.Assert().Error(err, "should error when connecting to database from url")
		switch osname := runtime.GOOS; osname {
		case "windows":
			suite.Assert().EqualError(err, "driver: sqlite, message: failed to open db file, originalError: CreateFile something invalid: The system cannot find the file specified. ")
		default:
			suite.Assert().EqualError(err, "driver: sqlite, message: failed to open db file, originalError: stat something invalid: no such file or directory ")
		}
	})

	suite.T().Run("when connURL is valid and bare uses default configuration", func(t *testing.T) {
		connectedDriver := suite.InitializeDriver(testConnURL)
		t.Cleanup(func() {
			require.NoError(t, connectedDriver.Close(), "should close the driver w/o errors")
		})

		cfg := getDefaultConfig()
		cfg.closeDBonClose = true // since we open the driver via DSN, we set closeDBonClose to true

		sqliteDriver := connectedDriver.(*sqlite)
		suite.Assert().EqualValues(cfg, sqliteDriver.config)
	})

}

func (suite *SqliteTestSuite) TestCreateSchemaTableIfNotExists() {
	suite.T().Run("it errors when connection is missing", func(t *testing.T) {
		driver := &sqlite{}

		_, err := driver.AppliedMigrations()
		suite.Assert().Error(err, "should error when database connection is missing")
		suite.Assert().EqualError(err, "driver: sqlite, message: database connection is missing, originalError: driver has no connection established ")
	})

	defaultConfig := getDefaultConfig()

	suite.T().Run("when x-migrations-table is missing, it creates a migrations table if not exists based on the default configuration", func(t *testing.T) {
		connectedDriver := suite.InitializeDriver(testConnURL)
		t.Cleanup(func() {
			require.NoError(t, connectedDriver.Close(), "should close the driver w/o errors")
		})

		driver, ok := connectedDriver.(*sqlite)
		suite.Require().True(ok)

		_, err := driver.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", defaultConfig.MigrationsTable))
		suite.Require().NoError(err, "should not error while dropping pre-existing migrations table")

		migrationTableExists := fmt.Sprintf(`SELECT COUNT(*) FROM sqlite_master
								WHERE  type = 'table'
								AND    name = '%s';`, defaultConfig.MigrationsTable)

		_, err = connectedDriver.AppliedMigrations()
		suite.Require().NoError(err, "should not error when creating the migrations table")

		var result int
		err = driver.db.QueryRow(migrationTableExists).Scan(&result)
		suite.Require().NoError(err, "should not error querying table existence")
		suite.Require().Equal(1, result, "migrations table should exist")
	})

	suite.T().Run("when x-migrations-table exists, it creates a migrations table if not exists", func(t *testing.T) {
		connectedDriver := suite.InitializeDriver(testConnURL + "?x-migrations-table=awesome_migrations")
		t.Cleanup(func() {
			require.NoError(t, connectedDriver.Close(), "should close the driver w/o errors")
		})

		driver, ok := connectedDriver.(*sqlite)
		suite.Require().True(ok)

		migrationTableExists := fmt.Sprintf(`SELECT COUNT(*) FROM sqlite_master
								WHERE  type = 'table'
								AND    name = '%s';`, "awesome_migrations")
		var result int
		err := driver.db.QueryRow(migrationTableExists).Scan(&result)
		suite.Require().NoError(err, "should not error querying table existence")
		suite.Require().Equal(0, result, "migrations table should not exist")

		_, err = connectedDriver.AppliedMigrations()
		suite.Require().NoError(err, "should not error when creating the migrations table")

		err = driver.db.QueryRow(migrationTableExists).Scan(&result)
		suite.Require().NoError(err, "should not error querying table existence")
		suite.Require().Equal(1, result, "migrations table should exist")
	})
}

func (suite *SqliteTestSuite) TestLock() {
	connectedDriver := suite.InitializeDriver(testConnURL)
	suite.T().Cleanup(func() {
		require.NoError(suite.T(), connectedDriver.Close(), "should close the driver w/o errors")
	})

	driver, ok := connectedDriver.(*sqlite)
	suite.Require().True(ok)

	err := driver.lock()
	suite.Require().NoError(err, "should not error when attempting to acquire a lock")
	defer func() { suite.Require().NoError(driver.unlock()) }()

	err = driver.lock()
	suite.Require().Error(err, "should error when attempting to acquire a lock if driver is locked")
}

func (suite *SqliteTestSuite) TestUnlock() {
	connectedDriver := suite.InitializeDriver(testConnURL)
	suite.T().Cleanup(func() {
		require.NoError(suite.T(), connectedDriver.Close(), "should close the driver w/o errors")
	})

	driver, ok := connectedDriver.(*sqlite)
	suite.Require().True(ok)

	err := driver.lock()
	suite.Require().NoError(err, "should not error when attempting to acquire a lock")

	err = driver.unlock()
	suite.Require().NoError(err, "should not error when attempting to release a lock")

	err = driver.unlock()
	suite.Require().NoError(err, "should not error when attempting to release an unlocked driver")
}

func (suite *SqliteTestSuite) TestAppliedMigrations() {
	connectedDriver := suite.InitializeDriver(testConnURL)
	suite.T().Cleanup(func() {
		require.NoError(suite.T(), connectedDriver.Close(), "should close the driver w/o errors")
	})

	_, err := connectedDriver.AppliedMigrations()
	suite.Require().NoError(err, "should not error when creating migrations table")

	defaultConfig := getDefaultConfig()

	insertMigrationsQuery := fmt.Sprintf(`
		INSERT INTO %s(Version, Name)
		VALUES
		       (1, 'test_1'),
			   (3, 'test_3'),
			   (2, 'test_2');
	`, defaultConfig.MigrationsTable)
	driver, ok := connectedDriver.(*sqlite)
	suite.Require().True(ok)
	_, err = driver.db.Exec(insertMigrationsQuery)
	suite.Require().NoError(err, "should not error when inserting seed migrations")
	appliedMigrations, err := connectedDriver.AppliedMigrations()
	suite.Require().NoError(err, "should not error when fetching applied migrations")
	suite.Assert().Len(appliedMigrations, 3)
	_, err = driver.db.Exec(fmt.Sprintf("DELETE FROM %s", defaultConfig.MigrationsTable))
	suite.Require().NoError(err, "should not error when deleting seed migrations")
}

func (suite *SqliteTestSuite) TestApply() {
	defaultConfig := getDefaultConfig()

	testData := []struct {
		Scenario                  string
		PendingMigrations         []*models.Migration
		AppliedMigrations         []*models.Migration
		ExpectedAppliedMigrations int
		Errors                    []error
	}{
		{
			"with no applied migrations and single statement, it applies migration",
			[]*models.Migration{
				{
					Version: 1,
					Bytes:   []byte("select 1;"),
					Name:    "migration_1.sql",
				},
			},
			[]*models.Migration{},
			1,
			[]error{nil},
		},
		{
			"with no applied migrations and multiple statements, it applies migration",
			[]*models.Migration{
				{
					Version: 1,
					Bytes:   []byte("select 1;\nselect 1;"),
					Name:    "migration_1.sql",
				},
			},
			[]*models.Migration{},
			1,
			[]error{nil},
		},
		{
			"with applied migrations and single statement, it applies migration",
			[]*models.Migration{
				{
					Version: 2,
					Bytes:   []byte("select 1;"),
					Name:    "migration_2.sql",
				},
			},
			[]*models.Migration{
				{
					Version: 1,
					Bytes:   []byte("select 1;"),
					Name:    "migration_1.sql",
				},
			},
			2,
			[]error{nil, nil},
		},
		{
			"when migration fails, it rollback the migration",
			[]*models.Migration{
				{
					Version: 1,
					Bytes:   []byte("select * from foobar;"),
					Name:    "migration_1.sql",
				},
			},
			[]*models.Migration{},
			0,
			[]error{
				errors.New("driver: sqlite, message: failed when applying migration, command: apply_migration, originalError: SQL logic error: no such table: foobar (1), query: \n\nselect * from foobar;\n"),
			},
		},
		{
			"when future migration fails, it rollback only the failed migration",
			[]*models.Migration{
				{
					Version: 1,
					Bytes:   []byte("select 1;"),
					Name:    "migration_1.sql",
				},
				{
					Version: 2,
					Bytes:   []byte("select * from foobar;"),
					Name:    "migration_2.sql",
				},
			},
			[]*models.Migration{},
			1,
			[]error{
				nil,
				errors.New("driver: sqlite, message: failed when applying migration, command: apply_migration, originalError: SQL logic error: no such table: foobar (1), query: \n\nselect * from foobar;\n"),
			},
		},
	}

	for _, elem := range testData {
		suite.T().Run(elem.Scenario, func(t *testing.T) {
			appliedMigrations := elem.AppliedMigrations
			pendingMigrations := elem.PendingMigrations
			expectedAppliedMigrations := elem.ExpectedAppliedMigrations
			expectedErrors := elem.Errors

			connectedDriver := suite.InitializeDriver(testConnURL)
			t.Cleanup(func() {
				require.NoError(t, connectedDriver.Close(), "should close the driver w/o errors")
			})

			driver, ok := connectedDriver.(*sqlite)
			suite.Require().True(ok)
			// Clear the migrations table
			_, err := driver.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", "db_migrations"))
			suite.Require().NoError(err, "should not error when dropping the test database")

			_, err = connectedDriver.AppliedMigrations()
			suite.Require().NoError(err, "should not error when creating migrations table")

			for _, appliedMigration := range appliedMigrations {
				insertMigrationsQuery := fmt.Sprintf(`
						INSERT INTO %s(Version, Name)
						VALUES
							   (%d, '%s');
					`, defaultConfig.MigrationsTable, appliedMigration.Version, appliedMigration.Name)
				_, err = driver.db.Exec(insertMigrationsQuery)
				suite.Require().NoError(err, "should not error when inserting seed migrations")
			}

			for i, pendingMigration := range pendingMigrations {
				err = connectedDriver.Apply(pendingMigration, true)
				if expectedErrors[i] != nil {
					suite.Assert().EqualErrorf(err, expectedErrors[i].Error(), "")
				} else {
					suite.Require().NoError(err, "should not error applying migration")
				}
			}

			var migrations int
			err = driver.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s;", defaultConfig.MigrationsTable)).Scan(&migrations)
			suite.Require().NoError(err, "should not error counting applied migrations")

			suite.Assert().Equal(expectedAppliedMigrations, migrations)

			_, err = driver.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", defaultConfig.MigrationsTable))
			suite.Require().NoError(err, "should not error while dropping migrations table")
		})
	}

}

func (suite *SqliteTestSuite) TestWithInstance() {
	db, err := sql.Open(driverName, testConnURL)
	suite.Require().NoError(err, "should not error when connecting to the test database")
	defer func() {
		err = db.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	}()
	suite.Assert().NoError(db.Ping(), "should not error when pinging the database")

	driver, err := WithInstance(db)
	sqliteDriver := driver.(*sqlite)
	sqliteDriver.config.closeDBonClose = true

	suite.Assert().NoError(err, "should not error when creating a driver from db instance")
	defer func() {
		err = driver.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	}()
}

func TestSqliteTestSuite(t *testing.T) {
	defaultDBFile, err := os.CreateTemp("", "morph-default.db")
	require.NoError(t, err)
	info, err := defaultDBFile.Stat()
	require.NoError(t, err)

	testDBFile, err := os.CreateTemp("", "morph-test.db")
	require.NoError(t, err)
	tfInfo, err := testDBFile.Stat()
	require.NoError(t, err)

	testConnURL = filepath.Join(os.TempDir(), info.Name())
	defaultConnURL = filepath.Join(os.TempDir(), tfInfo.Name())

	defer os.Remove(testConnURL)
	defer os.Remove(defaultConnURL)

	suite.Run(t, new(SqliteTestSuite))
}
