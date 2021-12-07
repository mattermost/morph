//go:build !sources && drivers
// +build !sources,drivers

package postgres

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/go-morph/morph/drivers"
	"github.com/go-morph/morph/models"
	"github.com/pkg/errors"

	"github.com/stretchr/testify/suite"
)

var (
	databaseName = "morph_test"
	testConnURL  = fmt.Sprintf("postgres://postgres:morph@localhost:5432/%s?sslmode=disable", databaseName)
)

const adminConnURL = "postgres://postgres:morph@localhost:5432?sslmode=disable"

type PostgresTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (suite *PostgresTestSuite) BeforeTest(_, _ string) {
	db, err := sql.Open(driverName, adminConnURL)
	suite.Require().NoError(err, "should not error when connecting as admin to the database")
	defer func() {
		err = db.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	}()

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName))
	suite.Require().NoError(err, "should not error when dropping the test database")

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", databaseName))
	suite.Require().NoError(err, "should not error when creating the test database")

	suite.db, err = sql.Open(driverName, testConnURL)
	suite.Require().NoError(err, "should not error when connecting to the test database")
}

func (suite *PostgresTestSuite) InitializeDriver(connURL string) (drivers.Driver, func()) {
	connectedDriver, err := Open(connURL)
	suite.Require().NoError(err, "should not error when connecting to database from url")
	suite.Require().NotNil(connectedDriver)

	return connectedDriver, func() {
		err = connectedDriver.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	}
}

func (suite *PostgresTestSuite) AfterTest(_, _ string) {
	if suite.db != nil {
		err := suite.db.Close()
		suite.Require().NoError(err, "should not error when closing the test database connection")
	}

	db, err := sql.Open(driverName, adminConnURL)
	suite.Require().NoError(err, "should not error when connecting as admin to the database")
	defer func() {
		err = db.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	}()

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName))
	suite.Require().NoError(err, "should not error when dropping the test database")
}

func (suite *PostgresTestSuite) TestOpen() {
	suite.T().Run("when connURL is valid and bare(no custom configuration present)", func(t *testing.T) {
		_, teardown := suite.InitializeDriver(testConnURL)
		defer teardown()
	})

	suite.T().Run("when connURL is invalid", func(t *testing.T) {
		_, err := Open("something invalid")
		suite.Assert().Error(err, "should error when connecting to database from url")
		suite.Assert().EqualError(err, "driver: postgres, message: failed to grab connection to the database, command: grabbing_connection, originalError: missing \"=\" after \"something\" in connection info string\", query: \n\n\n")
	})

	suite.T().Run("when connURL is valid and bare uses default configuration", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL)
		defer teardown()

		pgDriver := connectedDriver.(*postgres)
		suite.Assert().EqualValues(defaultConfig, pgDriver.config)
	})

	suite.T().Run("when connURL is valid can override migrations table", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL + "&x-migrations-table=test")
		defer teardown()

		pgDriver := connectedDriver.(*postgres)
		suite.Assert().Equal("test", pgDriver.config.MigrationsTable)
	})

	suite.T().Run("when connURL is valid can override statement timeout", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL + "&x-statement-timeout=10")
		defer teardown()

		pgDriver := connectedDriver.(*postgres)
		suite.Assert().Equal(10, pgDriver.config.StatementTimeoutInSecs)
	})

	suite.T().Run("when connURL is valid can override max migration size", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL + "&x-migration-max-size=42")
		defer teardown()

		pgDriver := connectedDriver.(*postgres)
		suite.Assert().Equal(42, pgDriver.config.MigrationMaxSize)
	})

	suite.T().Run("when connURL is valid extracts database and schema names", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL)
		defer teardown()

		pgDriver := connectedDriver.(*postgres)
		suite.Assert().Equal(databaseName, pgDriver.config.databaseName)
		suite.Assert().Equal("public", pgDriver.config.schemaName)
	})
}

func (suite *PostgresTestSuite) TestCreateSchemaTableIfNotExists() {
	suite.T().Run("it errors when connection is missing", func(t *testing.T) {
		driver := &postgres{}

		_, err := driver.AppliedMigrations()
		suite.Assert().Error(err, "should error when database connection is missing")
		suite.Assert().EqualError(err, "driver: postgres, message: database connection is missing, originalError: driver has no connection established ")
	})

	suite.T().Run("when x-migrations-table is missing, it creates a migrations table if not exists based on the default configuration", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL)
		defer teardown()

		_, err := suite.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS public.%s", defaultConfig.MigrationsTable))
		suite.Require().NoError(err, "should not error while dropping pre-existing migrations table")

		migrationTableExists := fmt.Sprintf(`SELECT COUNT(*) FROM pg_catalog.pg_class c
								JOIN   pg_catalog.pg_namespace n ON n.oid = c.relnamespace
								WHERE  n.nspname = 'public'
								AND    c.relname = '%s'
								AND    c.relkind = 'r';`, defaultConfig.MigrationsTable)
		_, err = connectedDriver.AppliedMigrations()
		suite.Require().NoError(err, "should not error when creating the migrations table")

		var result int
		err = suite.db.QueryRow(migrationTableExists).Scan(&result)
		suite.Require().NoError(err, "should not error querying table existence")
		suite.Require().Equal(1, result, "migrations table should exist")
	})

	suite.T().Run("when x-migrations-table exists, it creates a migrations table if not exists", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL)
		defer teardown()

		migrationTableExists := fmt.Sprintf(`SELECT COUNT(*) FROM pg_catalog.pg_class c
								JOIN   pg_catalog.pg_namespace n ON n.oid = c.relnamespace
								WHERE  n.nspname = 'public'
								AND    c.relname = '%s'
								AND    c.relkind = 'r';`, "awesome_migrations")
		var result int
		err := suite.db.QueryRow(migrationTableExists).Scan(&result)
		suite.Require().NoError(err, "should not error querying table existence")
		suite.Require().Equal(0, result, "migrations table should not exist")

		_, err = connectedDriver.AppliedMigrations()
		suite.Require().NoError(err, "should not error when creating the migrations table")

		migrationTableExists = fmt.Sprintf(`SELECT COUNT(*) FROM pg_catalog.pg_class c
								JOIN   pg_catalog.pg_namespace n ON n.oid = c.relnamespace
								WHERE  n.nspname = 'public'
								AND    c.relname = '%s'
								AND    c.relkind = 'r';`, defaultConfig.MigrationsTable)

		err = suite.db.QueryRow(migrationTableExists).Scan(&result)
		suite.Require().NoError(err, "should not error querying table existence")
		suite.Require().Equal(1, result, "migrations table should exist")
	})
}

func (suite *PostgresTestSuite) TestLock() {
	connectedDriver, teardown := suite.InitializeDriver(testConnURL)
	defer teardown()

	err := connectedDriver.Lock()
	suite.Require().NoError(err, "should not error when attempting to acquire an advisory lock")
	defer connectedDriver.Unlock()

	advisoryLockID, err := drivers.GenerateAdvisoryLockID("morph_test", "public")
	suite.Require().NoError(err, "should not error when generating generate advisory lock id")

	var result int
	err = suite.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM pg_locks WHERE locktype = 'advisory' AND granted = true AND objid = '%s'", advisoryLockID)).Scan(&result)
	suite.Require().NoError(err, "should not error querying pg_locks")
	suite.Require().Equal(1, result, "advisory lock should be acquired")
}

func (suite *PostgresTestSuite) TestUnlock() {
	connectedDriver, teardown := suite.InitializeDriver(testConnURL)
	defer teardown()

	err := connectedDriver.Lock()
	suite.Require().NoError(err, "should not error when attempting to acquire an advisory lock")

	advisoryLockID, err := drivers.GenerateAdvisoryLockID("morph_test", "public")
	suite.Require().NoError(err, "should not error when generating generate advisory lock id")

	err = connectedDriver.Unlock()
	suite.Require().NoError(err, "should not error when attempting to release an advisory lock")

	var result int
	err = suite.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM pg_locks WHERE locktype = 'advisory' AND granted = true AND objid = '%s'", advisoryLockID)).Scan(&result)
	suite.Require().NoError(err, "should not error querying pg_locks")
	suite.Require().Equal(0, result, "advisory lock should be released")
}

func (suite *PostgresTestSuite) TestAppliedMigrations() {
	connectedDriver, teardown := suite.InitializeDriver(testConnURL)
	defer teardown()

	_, err := connectedDriver.AppliedMigrations()
	suite.Require().NoError(err, "should not error when creating migrations table")

	insertMigrationsQuery := fmt.Sprintf(`
		INSERT INTO %s(version, name)
		VALUES
		       (1, 'test_1'),
			   (3, 'test_3'),
			   (2, 'test_2');
	`, defaultConfig.MigrationsTable)
	_, err = suite.db.Exec(insertMigrationsQuery)
	suite.Require().NoError(err, "should not error when inserting seed migrations")

	appliedMigrations, err := connectedDriver.AppliedMigrations()
	suite.Require().NoError(err, "should not error when fetching applied migrations")
	suite.Assert().Len(appliedMigrations, 3)
}

func (suite *PostgresTestSuite) TestApply() {
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
				errors.New("driver: postgres, message: failed to execute migration, command: executing_query, originalError: pq: relation \"foobar\" does not exist, query: \n\nselect * from foobar;\n"),
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
				errors.New("driver: postgres, message: failed to execute migration, command: executing_query, originalError: pq: relation \"foobar\" does not exist, query: \n\nselect * from foobar;\n"),
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

			_, err := connectedDriver.AppliedMigrations()
			suite.Require().NoError(err, "should not error when creating migrations table")
			defer func() {
				_, err = suite.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS public.%s", defaultConfig.MigrationsTable))
				suite.Require().NoError(err, "should not error while dropping migrations table")
			}()

			for _, appliedMigration := range appliedMigrations {
				insertMigrationsQuery := fmt.Sprintf(`
					INSERT INTO %s(version, name)
					VALUES
		       			(%d, '%s');
				`, defaultConfig.MigrationsTable, appliedMigration.Version, appliedMigration.Name)
				_, err = suite.db.Exec(insertMigrationsQuery)
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
			err = suite.db.QueryRow(fmt.Sprintf("select count(*) from %s;", defaultConfig.MigrationsTable)).Scan(&migrations)
			suite.Require().NoError(err, "should not error counting applied migrations")

			suite.Assert().Equal(expectedAppliedMigrations, migrations)
		})
	}
}

func (suite *PostgresTestSuite) TestWithInstance() {
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
	suite.Assert().Equal("public", config.schemaName)
}

func TestPostgresSuite(t *testing.T) {
	suite.Run(t, new(PostgresTestSuite))
}
