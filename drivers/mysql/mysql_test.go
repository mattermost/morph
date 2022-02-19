//go:build !sources && drivers
// +build !sources,drivers

package mysql

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/mattermost/morph/drivers"
	"github.com/mattermost/morph/models"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
)

var (
	databaseName   = "morph_test"
	testConnURL    = fmt.Sprintf("morph:morph@tcp(127.0.0.1:3307)/%s", databaseName)
	defaultConnURL = "root:morph@tcp(127.0.0.1:3307)/"
)

type MysqlTestSuite struct {
	suite.Suite
	db     *sql.DB
	testDB *sql.DB
}

func (suite *MysqlTestSuite) BeforeTest(_, _ string) {
	var err error
	suite.db, err = sql.Open(driverName, defaultConnURL)
	suite.Require().NoError(err, "should not error when connecting to the default database")

	suite.Require().NoError(suite.db.Ping())

	_, err = suite.db.Exec(`UPDATE performance_schema.setup_instruments
	SET ENABLED = 'YES', TIMED = 'YES'
	WHERE NAME = 'wait/lock/metadata/sql/mdl'`)
	suite.Require().NoError(err, "should not error when enabling granular lock telemetry")

	_, err = suite.db.Exec(`UPDATE performance_schema.setup_consumers SET ENABLED = 'YES' WHERE NAME = 'global_instrumentation'`)
	suite.Require().NoError(err, "should not error when enabling granular lock telemetry")

	_, err = suite.db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName))
	suite.Require().NoError(err, "should not error when dropping the test database")

	_, err = suite.db.Exec(fmt.Sprintf("CREATE DATABASE %s", databaseName))
	suite.Require().NoError(err, "should not error when creating the test database")

	suite.testDB, err = sql.Open(driverName, testConnURL)
	suite.Require().NoError(err, "should not error when connecting to the test database")

	suite.Require().NoError(suite.testDB.Ping())
}

func (suite *MysqlTestSuite) AfterTest(_, _ string) {
	_, err := suite.db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName))
	suite.Require().NoError(err, "should not error when dropping the test database")

	if suite.db != nil {
		err := suite.db.Close()
		suite.Require().NoError(err, "should not error when closing the default database connection")
	}

	if suite.testDB != nil {
		err := suite.testDB.Close()
		suite.Require().NoError(err, "should not error when closing the test database connection")
	}
}

func (suite *MysqlTestSuite) InitializeDriver(connURL string) (drivers.Driver, func()) {
	connectedDriver, err := Open(connURL)
	suite.Require().NoError(err, "should not error when connecting to database from url")
	suite.Require().NotNil(connectedDriver)

	return connectedDriver, func() {
		err = connectedDriver.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	}
}

func (suite *MysqlTestSuite) TestOpen() {
	suite.T().Run("when connURL is valid and bare(no custom configuration present)", func(t *testing.T) {
		_, teardown := suite.InitializeDriver(testConnURL)
		defer teardown()
	})

	suite.T().Run("when connURL is invalid", func(t *testing.T) {
		_, err := Open("something invalid")

		suite.Assert().Error(err, "should error when connecting to database from url")
		suite.Assert().EqualError(err, "driver: mysql, message: failed to open connection with the database, command: opening_connection, originalError: invalid DSN: missing the slash separating the database name, query: \n\n\n")
	})

	suite.T().Run("when connURL is valid and bare uses default configuration", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(defaultConnURL)
		defer teardown()

		defaultConfig := getDefaultConfig()
		cfg := &Config{
			Config: drivers.Config{
				MigrationsTable:        defaultConfig.MigrationsTable,
				StatementTimeoutInSecs: defaultConfig.StatementTimeoutInSecs,
				MigrationMaxSize:       defaultConfig.MigrationMaxSize,
			},
			closeDBonClose: true, // we have created DB from DSN
		}
		mysqlDriver := connectedDriver.(*mysql)
		suite.Assert().EqualValues(cfg, mysqlDriver.config)
	})

	suite.T().Run("when connURL is valid can override migrations table", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL + "?x-migrations-table=test")
		defer teardown()

		mysqlDriver := connectedDriver.(*mysql)
		suite.Assert().Equal("test", mysqlDriver.config.MigrationsTable)
	})

	suite.T().Run("when connURL is valid can override statement timeout", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL + "?x-statement-timeout=10")
		defer teardown()

		mysqlDriver := connectedDriver.(*mysql)
		suite.Assert().Equal(10, mysqlDriver.config.StatementTimeoutInSecs)
	})

	suite.T().Run("when connURL is valid can override max migration size", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL + "?x-migration-max-size=42")
		defer teardown()

		mysqlDriver := connectedDriver.(*mysql)
		suite.Assert().Equal(42, mysqlDriver.config.MigrationMaxSize)
	})

	suite.T().Run("when connURL is valid extracts database name", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL)
		defer teardown()

		mysqlDriver := connectedDriver.(*mysql)
		suite.Assert().Equal(databaseName, mysqlDriver.config.databaseName)
	})
}

func (suite *MysqlTestSuite) TestCreateSchemaTableIfNotExists() {
	defaultConfig := getDefaultConfig()

	suite.T().Run("it errors when connection is missing", func(t *testing.T) {
		driver := &mysql{}

		_, err := driver.AppliedMigrations()
		suite.Assert().Error(err, "should error when database connection is missing")
		suite.Assert().EqualError(err, "driver: mysql, message: database connection is missing, originalError: driver has no connection established ")
	})

	suite.T().Run("when x-migrations-table is missing, it creates a migrations table if not exists based on the default configuration", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL)
		defer teardown()

		_, err := suite.testDB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", defaultConfig.MigrationsTable))
		suite.Require().NoError(err, "should not error while dropping pre-existing migrations table")

		migrationTableExists := fmt.Sprintf(`SELECT COUNT(*) FROM information_schema.tables
								WHERE  table_schema = '%s'
								AND    table_name = '%s';`, databaseName, defaultConfig.MigrationsTable)

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

		migrationTableExists := fmt.Sprintf(`SELECT COUNT(*) FROM information_schema.tables
								WHERE  table_schema = '%s'
								AND    table_name = 'awesome_migrations';`, databaseName)
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

func (suite *MysqlTestSuite) TestUnlock() {
	defaultConfig := getDefaultConfig()

	connectedDriver, teardown := suite.InitializeDriver(testConnURL)
	defer teardown()

	err := connectedDriver.Lock()
	suite.Require().NoError(err, "should not error when attempting to acquire an advisory lock")

	advisoryLockID, err := drivers.GenerateAdvisoryLockID("morph_test", defaultConfig.MigrationsTable)
	suite.Require().NoError(err, "should not error when generating generate advisory lock id")

	err = connectedDriver.Unlock()
	suite.Require().NoError(err, "should not error when attempting to release an advisory lock")

	var result int
	err = suite.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM performance_schema.metadata_locks WHERE OBJECT_TYPE = 'USER LEVEL LOCK' AND LOCK_STATUS = 'GRANTED' AND OBJECT_NAME = '%s'", advisoryLockID)).Scan(&result)
	suite.Require().NoError(err, "should not error querying performance_schema.metadata_locks")
	suite.Require().Equal(0, result, "advisory lock should be released")
}

func (suite *MysqlTestSuite) TestAppliedMigrations() {
	defaultConfig := getDefaultConfig()

	connectedDriver, teardown := suite.InitializeDriver(testConnURL)
	defer teardown()

	_, err := connectedDriver.AppliedMigrations()
	suite.Require().NoError(err, "should not error when creating migrations table")

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

func (suite *MysqlTestSuite) TestApply() {
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
				errors.New("driver: mysql, message: failed when applying migration, command: apply_migration, originalError: Error 1146: Table 'morph_test.foobar' doesn't exist, query: \n\nselect * from foobar;\n"),
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
				errors.New("driver: mysql, message: failed when applying migration, command: apply_migration, originalError: Error 1146: Table 'morph_test.foobar' doesn't exist, query: \n\nselect * from foobar;\n"),
			},
		},
	}

	for _, elem := range testData {
		suite.T().Run(elem.Scenario, func(t *testing.T) {
			appliedMigrations := elem.AppliedMigrations
			pendingMigrations := elem.PendingMigrations
			expectedAppliedMigrations := elem.ExpectedAppliedMigrations
			expectedErrors := elem.Errors

			connectedDriver, teardown := suite.InitializeDriver(testConnURL + "?multiStatements=true")
			defer teardown()

			_, err := connectedDriver.AppliedMigrations()
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

func (suite *MysqlTestSuite) TestWithInstance() {
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

	suite.Assert().Equal(databaseName, config.databaseName)
}

func TestMysqlTestSuite(t *testing.T) {
	suite.Run(t, new(MysqlTestSuite))
}
