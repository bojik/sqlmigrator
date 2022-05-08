package migrator

import "text/template"

var goTemplate = template.Must(template.New("sqlmigrator.go").Parse(`package migrations

import (
	"github.com/bojik/sqlmigrator/pkg/migrator"
	"github.com/jmoiron/sqlx"
)

func init() {
	migrator.Exec.Add(
		func(db *sqlx.DB) (string, error) {
			return "", nil
		},
		func(db *sqlx.DB) (string, error) {
			return "", nil
		},
	)
}
`))
