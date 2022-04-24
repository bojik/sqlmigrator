package db

type Status int

//go:generate stringer -type=Status
const (
	New Status = iota + 1
	Success
	Error
)

//nolint:lll
//go:generate mockgen -destination=../mock/mock_data_keeper.go -package=mock github.com/bojik/sqlmigrator/pkg/db DataKeeper
type DataKeeper interface {
	CreateMigratorTable() error
	GetVersionsByStatus(Status) ([]int, error)
	FindNewMigrations(versions []int) ([]int, error)
	FindVersionStatusByID(version int) (Status, error)
	FindLastVersion() (int, error)
	DeleteByID(version int) error
	SelectVersionRow(int) (VersionRow, error)
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
