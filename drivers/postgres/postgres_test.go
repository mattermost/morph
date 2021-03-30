package postgres

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/go-morph/morph/drivers"

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

func (suite *PostgresTestSuite) SetupSuite() {
	db, err := sql.Open(driverName, adminConnURL)
	suite.Require().NoError(err, "should not error when connecting as admin to the database")

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName))
	suite.Require().NoError(err, "should not error when dropping the test database")

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", databaseName))
	suite.Require().NoError(err, "should not error when creating the test database")

	err = db.Close()
	suite.Require().NoError(err, "should not error when closing the database connection")

	suite.db, err = sql.Open(driverName, testConnURL)
	suite.Require().NoError(err, "should not error when connecting to the test database")
}

func (suite *PostgresTestSuite) TearDownSuite() {
	if suite.db != nil {
		err := suite.db.Close()
		suite.Require().NoError(err, "should not error when closing the test database connection")
	}

	db, err := sql.Open(driverName, adminConnURL)
	suite.Require().NoError(err, "should not error when connecting as admin to the database")

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName))
	suite.Require().NoError(err, "should not error when dropping the test database")

	err = db.Close()
	suite.Require().NoError(err, "should not error when closing the database connection")
}

func (suite *PostgresTestSuite) TestOpen() {
	suite.T().Run("when connURL is valid and bare(no custom configuration present)", func(t *testing.T) {
		driver, err := drivers.GetDriver(driverName)
		suite.Require().NoError(err, "fetching already registered driver should not fail")
		_, err = driver.Open(testConnURL)
		suite.Assert().NoError(err, "should not error when connecting to database from url")

		err = driver.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	})

	suite.T().Run("when connURL is invalid", func(t *testing.T) {
		driver, err := drivers.GetDriver(driverName)
		suite.Require().NoError(err, "fetching already registered driver should not fail")
		_, err = driver.Open("something invalid")
		suite.Assert().Error(err, "should error when connecting to database from url")
		suite.Assert().EqualError(err, "driver: postgres, message: failed to grab connection to the database, command: grabbing_connection, originalError: missing \"=\" after \"something%20invalid\" in connection info string\", query: ")
	})

	suite.T().Run("when connURL is valid and bare uses default configuration", func(t *testing.T) {
		driver, err := drivers.GetDriver(driverName)
		suite.Require().NoError(err, "fetching already registered driver should not fail")
		connectedDriver, err := driver.Open(testConnURL)
		suite.Assert().NoError(err, "should not error when connecting to database from url")

		pgDriver := connectedDriver.(*postgres)
		suite.Assert().EqualValues(defaultConfig, pgDriver.config)

		err = driver.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	})

	suite.T().Run("when connURL is valid can override migrations table", func(t *testing.T) {
		driver, err := drivers.GetDriver(driverName)
		suite.Require().NoError(err, "fetching already registered driver should not fail")
		connectedDriver, err := driver.Open(testConnURL + "&x-migrations-table=test")
		suite.Assert().NoError(err, "should not error when connecting to database from url")

		pgDriver := connectedDriver.(*postgres)
		suite.Assert().Equal("test", pgDriver.config.MigrationsTable)

		err = driver.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	})

	suite.T().Run("when connURL is valid can override statement timeout", func(t *testing.T) {
		driver, err := drivers.GetDriver(driverName)
		suite.Require().NoError(err, "fetching already registered driver should not fail")
		connectedDriver, err := driver.Open(testConnURL + "&x-statement-timeout=10")
		suite.Assert().NoError(err, "should not error when connecting to database from url")

		pgDriver := connectedDriver.(*postgres)
		suite.Assert().Equal(10, pgDriver.config.StatementTimeoutInSecs)

		err = driver.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	})

	suite.T().Run("when connURL is valid can override max migration size", func(t *testing.T) {
		driver, err := drivers.GetDriver(driverName)
		suite.Require().NoError(err, "fetching already registered driver should not fail")
		connectedDriver, err := driver.Open(testConnURL + "&x-migration-max-size=42")
		suite.Assert().NoError(err, "should not error when connecting to database from url")

		pgDriver := connectedDriver.(*postgres)
		suite.Assert().Equal(42, pgDriver.config.MigrationMaxSize)

		err = driver.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	})

	suite.T().Run("when connURL is valid extracts database and schema names", func(t *testing.T) {
		driver, err := drivers.GetDriver(driverName)
		suite.Require().NoError(err, "fetching already registered driver should not fail")
		connectedDriver, err := driver.Open(testConnURL)
		suite.Assert().NoError(err, "should not error when connecting to database from url")

		pgDriver := connectedDriver.(*postgres)
		suite.Assert().Equal(databaseName, pgDriver.config.DatabaseName)
		suite.Assert().Equal("public", pgDriver.config.SchemaName)

		err = driver.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	})
}

func TestPostgresSuite(t *testing.T) {
	suite.Run(t, new(PostgresTestSuite))
}
