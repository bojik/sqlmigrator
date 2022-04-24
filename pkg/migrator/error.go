package migrator

import "errors"

var (
	ErrIsNotDir                = errors.New("is not dir")
	ErrDownMigrationIsNotExist = errors.New("down migration is not exist")
	ErrUnfinishedMigrations    = errors.New("there is unfinished migrations")
	ErrVersionHasNotBeenFound  = errors.New("version has not been found")
)

type MigrationError struct {
	Err  error
	SQL  string
	File string
}

func NewMigrationError(err error, sql string, file string) MigrationError {
	return MigrationError{
		Err:  err,
		SQL:  sql,
		File: file,
	}
}

func (e MigrationError) Error() string {
	return e.Err.Error()
}

var _ error = (*MigrationError)(nil)
