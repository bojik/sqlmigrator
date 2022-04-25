package migrator

import "errors"

var (
	ErrIsNotDir                = errors.New("is not dir")
	ErrDownMigrationIsNotExist = errors.New("down migration is not exist")
	ErrUnfinishedMigrations    = errors.New("there is unfinished migrations")
	ErrVersionHasNotBeenFound  = errors.New("version has not been found")
)

type migrationError struct {
	Err  error
	SQL  string
	File string
}

func newMigrationError(err error, sql string, file string) migrationError {
	return migrationError{
		Err:  err,
		SQL:  sql,
		File: file,
	}
}

func (e migrationError) Error() string {
	return e.Err.Error()
}

var _ error = (*migrationError)(nil)
