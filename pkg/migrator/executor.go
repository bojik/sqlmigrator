package migrator

import (
	"runtime"

	"github.com/jmoiron/sqlx"
)

var Exec Executor

type MigrationFunc func(db *sqlx.DB) (string, error)

type goMigration struct {
	f    MigrationFunc
	file string
}

type goMigrations []goMigration

type Executor interface {
	Add(upFunc MigrationFunc, downFunc MigrationFunc)
	getUps() goMigrations
	getDowns() goMigrations
	reset()
}

func init() {
	Exec = &executor{}
}

type executor struct {
	ups   goMigrations
	downs goMigrations
}

func (e *executor) getUps() goMigrations {
	return e.ups
}

func (e *executor) getDowns() goMigrations {
	return e.downs
}

func (e *executor) reset() {
	e.ups = goMigrations{}
	e.downs = goMigrations{}
}

func (e *executor) Add(upFunc MigrationFunc, downFunc MigrationFunc) {
	//nolint:dogsled
	_, file, _, _ := runtime.Caller(1)
	e.ups = append(e.ups, goMigration{
		f:    upFunc,
		file: file,
	})
	e.downs = append(e.downs, goMigration{
		f:    downFunc,
		file: file,
	})
}

var _ Executor = (*executor)(nil)
