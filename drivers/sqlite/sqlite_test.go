//go:build !sources && drivers
// +build !sources,drivers

package sqlite

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/go-morph/morph/drivers"
	"github.com/stretchr/testify/suite"
)

var (
	databaseName   = "morph_test"
	testConnURL    = fmt.Sprintf("sqlite:/tmp/morph/db)/%s", databaseName)
	defaultConnURL = "sqlite:/tmp/morph/db"
)

type SqliteTestSuite struct {
	suite.Suite
	db     *sql.DB
	testDB *sql.DB
}

func (suite *SqliteTestSuite) BeforeTest(_, _ string) {
}

func (suite *SqliteTestSuite) AfterTest(_, _ string) {
}

func (suite *SqliteTestSuite) InitializeDriver(connURL string) (drivers.Driver, func()) {
	return nil, func() {}
}

func (suite *SqliteTestSuite) TestOpen() {
}

func (suite *SqliteTestSuite) TestCreateSchemaTableIfNotExists() {

}

func (suite *SqliteTestSuite) TestLock() {

}

func (suite *SqliteTestSuite) TestUnlock() {

}

func (suite *SqliteTestSuite) TestAppliedMigrations() {

}

func (suite *SqliteTestSuite) TestApply() {

}

func (suite *SqliteTestSuite) TestWithInstance() {

}

func TestMysqlTestSuite(t *testing.T) {
	suite.Run(t, new(SqliteTestSuite))
}
