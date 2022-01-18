package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-morph/morph/drivers"
	"github.com/go-morph/morph/models"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	testConnURL    = "morph-test.db"
	defaultConnURL = "morph-default.db"
)

type SqliteTestSuite struct {
	suite.Suite
	db     *sql.DB
	testDB *sql.DB
}

func (suite *SqliteTestSuite) BeforeTest(_, _ string) {
	var err error
	suite.db, err = sql.Open(driverName, defaultConnURL)
	suite.Require().NoError(err, "should not error when connecting to the default database")

	suite.Require().NoError(suite.db.Ping())

	suite.testDB, err = sql.Open(driverName, testConnURL)
	suite.Require().NoError(err, "should not error when connecting to the test database")

	suite.Require().NoError(suite.testDB.Ping())
}

func (suite *SqliteTestSuite) AfterTest(_, _ string) {
}

func (suite *SqliteTestSuite) InitializeDriver(connURL string) (drivers.Driver, func()) {
	connectedDriver, err := Open(connURL)
	suite.Require().NoError(err, "should not error when connecting to database from url")
	suite.Require().NotNil(connectedDriver)

	return connectedDriver, func() {
		err = connectedDriver.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	}
}

func (suite *SqliteTestSuite) TestOpen() {
	suite.T().Run("when connURL is valid and bare(no custom configuration present)", func(t *testing.T) {
		_, teardown := suite.InitializeDriver(testConnURL)
		defer teardown()
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
		connectedDriver, teardown := suite.InitializeDriver(testConnURL)
		defer teardown()

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
		connectedDriver, teardown := suite.InitializeDriver(testConnURL)
		defer teardown()

		_, err := suite.testDB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", defaultConfig.MigrationsTable))
		suite.Require().NoError(err, "should not error while dropping pre-existing migrations table")

		migrationTableExists := fmt.Sprintf(`SELECT COUNT(*) FROM sqlite_master
								WHERE  type = 'table'
								AND    name = '%s';`, defaultConfig.MigrationsTable)

		_, err = connectedDriver.AppliedMigrations()
		suite.Require().NoError(err, "should not error when creating the migrations table")

		var result int
		err = suite.testDB.QueryRow(migrationTableExists).Scan(&result)
		suite.Require().NoError(err, "should not error querying table existence")
		suite.Require().Equal(1, result, "migrations table should exist")
	})

	suite.T().Run("when x-migrations-table exists, it creates a migrations table if not exists", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL + "?x-migrations-table=awesome_migrations")
		defer teardown()

		migrationTableExists := fmt.Sprintf(`SELECT COUNT(*) FROM sqlite_master
								WHERE  type = 'table'
								AND    name = '%s';`, "awesome_migrations")
		var result int
		err := suite.testDB.QueryRow(migrationTableExists).Scan(&result)
		suite.Require().NoError(err, "should not error querying table existence")
		suite.Require().Equal(0, result, "migrations table should not exist")

		_, err = connectedDriver.AppliedMigrations()
		suite.Require().NoError(err, "should not error when creating the migrations table")

		err = suite.testDB.QueryRow(migrationTableExists).Scan(&result)
		suite.Require().NoError(err, "should not error querying table existence")
		suite.Require().Equal(1, result, "migrations table should exist")
	})
}

func (suite *SqliteTestSuite) TestLock() {

}

func (suite *SqliteTestSuite) TestUnlock() {

}

func (suite *SqliteTestSuite) TestAppliedMigrations() {
	connectedDriver, teardown := suite.InitializeDriver(testConnURL)
	defer teardown()
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
	_, err = suite.testDB.Exec(insertMigrationsQuery)
	suite.Require().NoError(err, "should not error when inserting seed migrations")
	appliedMigrations, err := connectedDriver.AppliedMigrations()
	suite.Require().NoError(err, "should not error when fetching applied migrations")
	suite.Assert().Len(appliedMigrations, 3)
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
					Bytes:   ioutil.NopCloser(strings.NewReader("select 1;")),
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
					Bytes:   ioutil.NopCloser(strings.NewReader("select 1;\nselect 1;")),
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
					Bytes:   ioutil.NopCloser(strings.NewReader("select 1;")),
					Name:    "migration_2.sql",
				},
			},
			[]*models.Migration{
				{
					Version: 1,
					Bytes:   ioutil.NopCloser(strings.NewReader("select 1;")),
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
					Bytes:   ioutil.NopCloser(strings.NewReader("select * from foobar;")),
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
					Bytes:   ioutil.NopCloser(strings.NewReader("select 1;")),
					Name:    "migration_1.sql",
				},
				{
					Version: 2,
					Bytes:   ioutil.NopCloser(strings.NewReader("select * from foobar;")),
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

			connectedDriver, teardown := suite.InitializeDriver(testConnURL)
			defer teardown()

			// Clear the migrations table
			_, err := suite.testDB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", "db_migrations"))
			suite.Require().NoError(err, "should not error when dropping the test database")

			_, err = connectedDriver.AppliedMigrations()
			suite.Require().NoError(err, "should not error when creating migrations table")

			for _, appliedMigration := range appliedMigrations {
				insertMigrationsQuery := fmt.Sprintf(`
						INSERT INTO %s(Version, Name)
						VALUES
							   (%d, '%s');
					`, defaultConfig.MigrationsTable, appliedMigration.Version, appliedMigration.Name)
				_, err = suite.testDB.Exec(insertMigrationsQuery)
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
			err = suite.testDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s;", defaultConfig.MigrationsTable)).Scan(&migrations)
			suite.Require().NoError(err, "should not error counting applied migrations")

			suite.Assert().Equal(expectedAppliedMigrations, migrations)

			_, err = suite.testDB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", defaultConfig.MigrationsTable))
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

	config := &Config{
		closeDBonClose: true,
	}
	driver, err := WithInstance(db, config)
	suite.Assert().NoError(err, "should not error when creating a driver from db instance")
	defer func() {
		err = driver.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	}()
}

func TestSqliteTestSuite(t *testing.T) {
	defaultDBFile, err := ioutil.TempFile("", "morph-default.db")
	require.NoError(t, err)
	info, err := defaultDBFile.Stat()
	require.NoError(t, err)

	testDBFile, err := ioutil.TempFile("", "morph-test.db")
	require.NoError(t, err)
	tfInfo, err := testDBFile.Stat()
	require.NoError(t, err)

	testConnURL = filepath.Join(os.TempDir(), info.Name())
	defaultConnURL = filepath.Join(os.TempDir(), tfInfo.Name())

	defer os.Remove(testConnURL)
	defer os.Remove(defaultConnURL)

	suite.Run(t, new(SqliteTestSuite))
}
