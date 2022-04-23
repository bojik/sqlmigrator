package migrator

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

const Dsn = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

var migrationDir = path.Join(testDataDir, "migrations")

func TestMigrator_ApplyUpSqlMigration(t *testing.T) {
	migrator := New(newEmptyLogger())
	err := migrator.ApplyUpSqlMigration(context.Background(), Dsn, migrationDir)
	require.Nil(t, err)
}

func TestFindFiles(t *testing.T) {
	files, err := findFiles(migrationDir, ".up.sql")
	require.Nil(t, err)
	fmt.Println(files)
}
