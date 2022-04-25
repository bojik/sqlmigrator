package db

import "time"

type Status int

//go:generate stringer -type=Status
const (
	Processing Status = iota + 1
	Success
	Error
)

type Row struct {
	Version int
	Status
	ExecutedAt time.Time
}

//nolint:lll
//go:generate mockgen -destination=../mock/mock_data_keeper.go -package=mock github.com/bojik/sqlmigrator/pkg/db DataKeeper
type DataKeeper interface {
	CreateMigratorTable() error
	GetVersionsByStatus(Status) ([]int, error)
	FindNewMigrations(versions []int) ([]int, error)
	FindVersionStatusByVersion(version int) (Status, error)
	FindLastVersion() (int, error)
	DeleteByVersion(version int) error
	SelectVersionRow(int) (VersionRow, error)
	SelectRows() ([]Row, error)
	ExecSQL(sql string) error
}

//nolint:lll
//go:generate mockgen -destination=../mock/mock_version_row.go -package=mock github.com/bojik/sqlmigrator/pkg/db VersionRow
type VersionRow interface {
	Close() error
	Insert() error
	Commit() error
	SetStatus(Status) error
	CommitSuccess() error
	CommitError() error
}
