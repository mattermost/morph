//go:build !sources && drivers
// +build !sources,drivers

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/mattermost/morph/drivers"
	"github.com/mattermost/morph/models"
	"github.com/pkg/errors"

	"github.com/stretchr/testify/suite"
)

var (
	databaseName = "morph_test"
	testConnURL  = fmt.Sprintf("postgres://morph:morph@localhost:6432/%s?sslmode=disable", databaseName)
)

const adminConnURL = "postgres://morph:morph@localhost?sslmode=disable"

type PostgresTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (suite *PostgresTestSuite) BeforeTest(_, _ string) {
	var err error
	suite.db, err = sql.Open(driverName, testConnURL)
	suite.Require().NoError(err, "should not error when connecting to the test database")

	_, err = suite.db.Exec(`DO $$ DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
    END LOOP;
END $$;`)
	suite.Require().NoError(err, "should not error when dropping the tables the test database")
}

func (suite *PostgresTestSuite) InitializeDriver(connURL string) (*Postgres, func()) {
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
		var err error
		suite.db, err = sql.Open(driverName, testConnURL)
		suite.Require().NoError(err, "should not error when connecting to the test database")

		_, err = suite.db.Exec(`DO $$ DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
    END LOOP;
END $$;`)
		suite.Require().NoError(err, "should not error when dropping the tables the test database")

		err = suite.db.Close()
		suite.Require().NoError(err, "should not error when closing the test database connection")
	}
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

		defaultConfig := getDefaultConfig()
		cfg := &driverConfig{
			Config: drivers.Config{
				MigrationsTable:        defaultConfig.MigrationsTable,
				StatementTimeoutInSecs: defaultConfig.StatementTimeoutInSecs,
				MigrationMaxSize:       defaultConfig.MigrationMaxSize,
			},
			databaseName:   databaseName,
			schemaName:     "public",
			closeDBonClose: true, // we have created DB from DSN
		}

		suite.Assert().EqualValues(cfg, connectedDriver.config)
	})

	suite.T().Run("when connURL is valid can override migrations table", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL + "&x-migrations-table=test")
		defer teardown()

		suite.Assert().Equal("test", connectedDriver.config.MigrationsTable)
	})

	suite.T().Run("when connURL is valid can override statement timeout", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL + "&x-statement-timeout=10")
		defer teardown()

		suite.Assert().Equal(10, connectedDriver.config.StatementTimeoutInSecs)
	})

	suite.T().Run("when connURL is valid can override max migration size", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL + "&x-migration-max-size=42")
		defer teardown()

		suite.Assert().Equal(42, connectedDriver.config.MigrationMaxSize)
	})

	suite.T().Run("when connURL is valid extracts database and schema names", func(t *testing.T) {
		connectedDriver, teardown := suite.InitializeDriver(testConnURL)
		defer teardown()

		suite.Assert().Equal(databaseName, connectedDriver.config.databaseName)
		suite.Assert().Equal("public", connectedDriver.config.schemaName)
	})
}

func (suite *PostgresTestSuite) TestCreateSchemaTableIfNotExists() {
	defaultConfig := getDefaultConfig()

	suite.T().Run("it errors when connection is missing", func(t *testing.T) {
		driver := &Postgres{}

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

func (suite *PostgresTestSuite) TestAppliedMigrations() {
	defaultConfig := getDefaultConfig()

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

func (suite *PostgresTestSuite) TestNonTransactionalMigrations() {
	connectedDriver, teardown := suite.InitializeDriver(testConnURL)
	defer teardown()

	err := connectedDriver.Apply(&models.Migration{
		Version: uint32(1),
		Name:    "create table",
		RawName: "test.sql",
		Bytes: []byte(`CREATE TABLE IF NOT EXISTS testtable (
				duedate bigint,
				completed boolean,
			    PRIMARY KEY (duedate)
			);
		`),
		Direction: models.Up,
	}, false)
	suite.Require().NoError(err, "should not error while creating a table")
	err = connectedDriver.Apply(&models.Migration{
		Version: uint32(2),
		Name:    "create index",
		RawName: "test.sql",
		Bytes: []byte(`-- morph:nontransactional
			CREATE INDEX CONCURRENTLY test on testtable(duedate);
		`),
		Direction: models.Up,
	}, false)
	suite.Require().NoError(err, "show not error while running a non-transactional migration")
}

func (suite *PostgresTestSuite) TestApply() {
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
				errors.New("driver: postgres, message: failed to execute migration, command: executing_query, originalError: pq: relation \"foobar\" does not exist, query: \n\nselect * from foobar;\n"),
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

	psqlDriver, err := WithInstance(db)
	psqlDriver.config.closeDBonClose = true
	suite.Assert().NoError(err, "should not error when creating a driver from db instance")
	defer func() {
		err = psqlDriver.Close()
		suite.Require().NoError(err, "should not error when closing the database connection")
	}()

	suite.Assert().Equal(databaseName, psqlDriver.config.databaseName)
	suite.Assert().Equal("public", psqlDriver.config.schemaName)
}

func TestPostgresSuite(t *testing.T) {
	suite.Run(t, new(PostgresTestSuite))
}

func (suite *PostgresTestSuite) TestLock() {
	connectedDriver, teardown := suite.InitializeDriver(testConnURL)
	defer teardown()

	logger := log.New(os.Stderr, "", 0)

	suite.T().Run("should create lock and unlock the mutex", func(t *testing.T) {
		ctx := context.Background()

		mx, err := connectedDriver.NewMutex("test-lock-key", logger)
		suite.Require().NoError(err, "should not error while creating the mutex")

		err = mx.Lock(ctx)
		suite.Require().NoError(err, "should not error while locking the mutex")

		err = mx.Unlock()
		suite.Require().NoError(err, "should not error while unlocking the mutex")
	})

	suite.T().Run("should release the expired lock", func(t *testing.T) {
		ctx := context.Background()

		query := fmt.Sprintf("INSERT INTO %s (id, expireat) VALUES ($1, $2)", drivers.MutexTableName)
		_, err := connectedDriver.conn.ExecContext(ctx, query, "test-lock-key", 1)
		suite.Require().NoError(err, "should not error while manually inserting the mutex")

		mx, err := connectedDriver.NewMutex("test-lock-key", logger)
		suite.Require().NoError(err, "should not error while creating the mutex")

		err = mx.Lock(ctx)
		suite.Require().NoError(err, "should not error while locking the mutex")

		err = mx.Unlock()
		suite.Require().NoError(err, "should not error while unlocking the mutex")
	})

	suite.T().Run("should refresh the lock after expired", func(t *testing.T) {
		ctx := context.Background()

		now := time.Now()
		timeout := time.After(2 * drivers.TTL) // should not wait to drop the lock for 30s

		query := fmt.Sprintf("INSERT INTO %s (id, expireat) VALUES ($1, $2)", drivers.MutexTableName)
		// set expiration 2 seconds later
		_, err := connectedDriver.conn.ExecContext(ctx, query, "test-lock-key", now.Add(2*time.Second).Unix())
		suite.Require().NoError(err, "should not error while manually inserting the mutex")

		done := make(chan struct{})
		go func() {
			defer func() {
				close(done)
			}()
			mx, err := connectedDriver.NewMutex("test-lock-key", logger)
			suite.Require().NoError(err, "should not error while creating the mutex")

			err = mx.Lock(ctx)
			suite.Require().NoError(err, "should not error while locking the mutex")

			// ensure we waited the lock to be expire
			suite.Require().True(time.Now().After(now.Add(1 * time.Second)))

			err = mx.Unlock()
			suite.Require().NoError(err, "should not error while unlocking the mutex")
		}()

		select {
		case <-timeout:
			suite.Require().Fail("should have wait and release the lock")
		case <-done:
		}
	})
}
