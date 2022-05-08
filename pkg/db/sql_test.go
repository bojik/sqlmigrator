package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSqlTemplate(t *testing.T) {
	s := sqlTemplate("{{table}}")
	require.Equal(t, "", s.sql(sqlTable("")))
	s = sqlTemplate("select {{table}}")
	require.Equal(t, "select test", s.sql(sqlTable("test")))
}
