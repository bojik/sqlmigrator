//go:build integration
// +build integration

package migrator

import (
	"context"
	"testing"

	"github.com/bojik/sqlmigrator/pkg/config"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"

	// postgresql driver.
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type MigratorTestSuite struct {
	suite.Suite
	db *sqlx.DB
}

var (
	dsn          string
	migrationDir string
)

func (s *MigratorTestSuite) SetupSuite() {
	db, err := sqlx.Open("postgres", dsn)
	require.Nil(s.T(), err)
	s.db = db
	_, err1 := db.Exec("drop table if exists test")
	require.Nil(s.T(), err1)
	_, err2 := db.Exec("drop table if exists test1")
	require.Nil(s.T(), err2)
	_, err3 := db.Exec("drop table if exists dbmigrator_version")
	require.Nil(s.T(), err3)
}

func (s *MigratorTestSuite) TearDownSuite() {
	err := s.db.Close()
	require.Nil(s.T(), err)
}

func (s *MigratorTestSuite) TestMigrator() {
	migrator := New(newEmptyLogger())
	_, err := migrator.ApplyUpSQLMigration(context.Background(), dsn, migrationDir)
	require.Nil(s.T(), err)
	rows, err := s.db.Query("select test from test")
	require.Nil(s.T(), err)
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	r := []string{}
	for rows.Next() {
		var v string
		err := rows.Scan(&v)
		require.Nil(s.T(), err)
		r = append(r, v)
	}
	require.Len(s.T(), r, 3)
	_, err4 := migrator.ApplyDownSQLMigration(context.Background(), dsn, migrationDir)
	require.Nil(s.T(), err4)
	res, err5 := migrator.ApplyRedoSQLMigration(context.Background(), dsn, migrationDir)
	require.Nil(s.T(), err5)
	require.Len(s.T(), res, 2)
	require.Equal(s.T(), res[0].Version, res[1].Version)
}

func TestFindFiles(t *testing.T) {
	loadConfig(t)
	files, err := findFiles(migrationDir, ".up.sql")
	require.Nil(t, err)
	require.Len(t, files, 5)
}

func TestMigratorTestSuite(t *testing.T) {
	loadConfig(t)
	suite.Run(t, new(MigratorTestSuite))
}

func loadConfig(t *testing.T) {
	err := config.Load("./testdata/config.yaml", &pflag.FlagSet{})
	require.Nil(t, err)
	dsn = config.GetDsn()
	migrationDir = config.GetPath()
}
