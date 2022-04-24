//go:build integration
// +build integration

package migrator

import (
	"context"
	"path"
	"testing"

	"github.com/jmoiron/sqlx"
	// postgresql driver.
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const Dsn = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

var migrationDir = path.Join(testDataDir, "migrations")

func TestMigrator_ApplyUpSqlMigration(t *testing.T) {
	migrator := New(newEmptyLogger())
	err := migrator.ApplyUpSQLMigration(context.Background(), Dsn, migrationDir)
	require.Nil(t, err)
}

func TestMigrator_ApplyDownSqlMigration(t *testing.T) {
	migrator := New(newEmptyLogger())
	err := migrator.ApplyDownSQLMigration(context.Background(), Dsn, migrationDir)
	require.Nil(t, err)
}

func TestMigratorResult(t *testing.T) {
	db, err := sqlx.Open("postgres", Dsn)
	rows, err := db.Query("select test from test")
	require.Nil(t, err)
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	r := []string{}
	for rows.Next() {
		var v string
		err := rows.Scan(&v)
		require.Nil(t, err)
		r = append(r, v)
	}
	require.Len(t, r, 3)
}

func TestFindFiles(t *testing.T) {
	files, err := findFiles(migrationDir, ".up.sql")
	require.Nil(t, err)
	require.Len(t, files, 5)
}
