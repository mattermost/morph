package morph

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"text/template"
	"time"

	"github.com/mattermost/morph/drivers"
	"github.com/mattermost/morph/drivers/mysql"
	"github.com/mattermost/morph/drivers/postgres"
	"github.com/mattermost/morph/drivers/sqlite"
	"github.com/mattermost/morph/models"
	"github.com/mattermost/morph/sources"
	"github.com/mattermost/morph/testlib"
	"github.com/stretchr/testify/require"
)

const (
	defaultPostgresDSN = "postgres://morph:morph@localhost:6432/morph_test?sslmode=disable"
	defaultMySQLDSN    = "morph:morph@tcp(127.0.0.1:3307)/morph_test?multiStatements=true"
)

// query is a map of driver name to a map of direction for the dummy queries
var queries = map[string]map[models.Direction]string{
	"postgres": {
		models.Up:   `CREATE TABLE IF NOT EXISTS {{.Name}} (id serial PRIMARY KEY, name text)`,
		models.Down: `DROP TABLE IF EXISTS {{.Name}}`,
	},
	"mysql": {
		models.Up:   `CREATE TABLE IF NOT EXISTS {{.Name}} (id int(11) NOT NULL AUTO_INCREMENT, name varchar(255), PRIMARY KEY (id))`,
		models.Down: `DROP TABLE IF EXISTS {{.Name}}`,
	},
	"sqlite": {
		models.Up:   `CREATE TABLE IF NOT EXISTS {{.Name}} (id integer PRIMARY KEY AUTOINCREMENT, name text)`,
		models.Down: `DROP TABLE IF EXISTS {{.Name}}`,
	},
}

// testHelper is a helper struct for testing morph engine.
// It contains all the necessary information to run tests for all drivers.
// It also provides helper functions to create dummy migrations.
type testHelper struct {
	drivers     map[string]drivers.Driver
	dbInstances map[string]*sql.DB
	sqliteFile  string
	options     []EngineOption
	migrations  map[string][]*models.Migration
}

// testSource is a dummy source for testing purposes.
type testSource struct {
	migrations []*models.Migration
}

func (s *testSource) Migrations() []*models.Migration {
	return s.migrations
}

// source returns a dummy source for the given driver
func (h *testHelper) source(driverName string) sources.Source {
	src := &testSource{
		migrations: h.migrations[driverName],
	}

	return src
}

func newTestHelper(t *testing.T, options ...EngineOption) *testHelper {
	helper := &testHelper{
		options:     options,
		drivers:     map[string]drivers.Driver{},
		migrations:  map[string][]*models.Migration{},
		dbInstances: map[string]*sql.DB{},
	}

	helper.initializeDrivers(t)

	return helper
}

// creates 3 new migrations
func (h *testHelper) CreateBasicMigrations(t *testing.T) *testHelper {
	h.AddMigration(t, "create_table_1")
	h.AddMigration(t, "create_table_2")
	h.AddMigration(t, "create_table_3")

	return h
}

// AddMigration adds a dummy migration to the test helper. It is important to add
// migrations before running the RunForAllDrivers function as migrations are registered
// before the test function is run.
func (h *testHelper) AddMigration(t *testing.T, migrationName string) {
	// Just generate a random name
	tableName := fmt.Sprintf("test_%s_%d", migrationName, time.Now().Unix())
	for name := range h.drivers {
		v := 1 + uint32(len(h.migrations[name]))
		h.migrations[name] = append(h.migrations[name], &models.Migration{
			Name:      migrationName,
			Direction: models.Up,
			Version:   v,
			Bytes:     getMigration(t, name, models.Up, tableName),
			RawName:   fmt.Sprintf("%d_%s.up.sql", v, migrationName),
		})
		h.migrations[name] = append(h.migrations[name], &models.Migration{
			Name:      migrationName,
			Direction: models.Down,
			Version:   v,
			Bytes:     getMigration(t, name, models.Down, tableName),
			RawName:   fmt.Sprintf("%d_%s.down.sql", v, migrationName),
		})
	}
}

// getMigration returns a dummy migration for the given driver and direction
func getMigration(t *testing.T, driver string, direction models.Direction, tableName string) []byte {
	tmp, err := template.New("query").Parse(queries[driver][direction])
	require.NoError(t, err)

	var b bytes.Buffer
	err = tmp.Execute(&b, struct{ Name string }{Name: tableName})
	require.NoError(t, err)

	return b.Bytes()
}

// RunForAllDrivers runs the given test function for all drivers of the test helper
func (h *testHelper) RunForAllDrivers(t *testing.T, f func(*testing.T, *Morph), name ...string) {
	var testName string
	if len(name) > 0 {
		testName = name[0] + "/"
	}

	for name, driver := range h.drivers {
		t.Run(testName+name, func(t *testing.T) {
			engine, err := New(context.Background(), driver, h.source(name), h.options...)
			require.NoError(t, err)

			f(t, engine)
		})
	}
}

// TearDown closes all database connections and removes all tables from the databases
func (h *testHelper) Teardown(t *testing.T) {
	assets := testlib.Assets()
	for name, driver := range h.drivers {
		b, err := assets.ReadFile(filepath.Join("scripts", name+"_drop_all_tables.sql"))
		require.NoError(t, err)
		migration := &models.Migration{
			Bytes: b,
		}
		err = driver.Apply(migration, false)
		require.NoError(t, err)
	}

	for _, instance := range h.dbInstances {
		err := instance.Close()
		require.NoError(t, err)
	}

	err := os.RemoveAll(h.sqliteFile)
	require.NoError(t, err)
}

func (h *testHelper) initializeDrivers(t *testing.T) {
	// postgres
	db, err := sql.Open("postgres", defaultPostgresDSN)
	require.NoError(t, err)

	pgDriver, err := postgres.WithInstance(db)
	require.NoError(t, err)
	h.drivers["postgres"] = pgDriver
	h.dbInstances["postgres"] = db

	// mysql
	db2, err := sql.Open("mysql", defaultMySQLDSN)
	require.NoError(t, err)

	mysqlDriver, err := mysql.WithInstance(db2)
	require.NoError(t, err)
	h.drivers["mysql"] = mysqlDriver
	h.dbInstances["mysql"] = db2

	// sqlite
	testDBFile, err := os.CreateTemp("", "morph-test.db")
	require.NoError(t, err)
	tfInfo, err := testDBFile.Stat()
	require.NoError(t, err)
	h.sqliteFile = filepath.Join(os.TempDir(), tfInfo.Name())

	db3, err := sql.Open("sqlite", h.sqliteFile)
	require.NoError(t, err)

	sqliteDriver, err := sqlite.WithInstance(db3)
	require.NoError(t, err)
	h.drivers["sqlite"] = sqliteDriver
	h.dbInstances["sqlite"] = db3
}
