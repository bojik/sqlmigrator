package db

import (
	"context"
	"fmt"
	"io"

	"github.com/jmoiron/sqlx"
	// postgresql driver.
	_ "github.com/lib/pq"
)

type Postgres struct {
	db  *sqlx.DB
	ctx context.Context
	log io.Writer
}

func NewPostgres(ctx context.Context, log io.Writer) *Postgres {
	return &Postgres{
		ctx: ctx,
		log: log,
	}
}

//nolint:lll
const (
	SQLCreateTable         = `create table if not exists dbmigrator_version (version bigint not null primary key, status smallint not null, executed_at timestamp with time zone);`
	SQLCreateTableIndex    = `create index if not exists status_idx on dbmigrator_version(status);create index if not exists executed_at_idx on dbmigrator_version(executed_at);`
	SQLCreateTableComments = `comment on table dbmigrator_version is 'dbmigrator migrations';
			comment on column dbmigrator_version.version is 'migration id';
			comment on column dbmigrator_version.status is '1 - new, 2 - successful finished, 3 - executed with error';
			comment on column dbmigrator_version.executed_at is 'date of the last executing attempt';`
	SQLSelectNewMigrations = `select version from dbmigrator_version where version in (?) and status <> ?`
	SQLInsert              = `insert into dbmigrator_version(version, status) values(:id, :status) on conflict do nothing`
	SQLSelectLock          = `select version from dbmigrator_version where version = :id for update`
	SQLUpdateStatus        = `update dbmigrator_version set status = :status, executed_at = current_timestamp where version = :id`
	SQLSelectByStatus      = `select version from dbmigrator_version where status = :status order by version`
	SQLSelectStatusByID    = `select status from dbmigrator_version where version = :version`
	SQLSelectLastVersion   = `select version from dbmigrator_version where executed_at = (select max(executed_at) from dbmigrator_version)`
	SQLDeleteByID          = `delete from dbmigrator_version where version = :version`
	SQLSelectAll           = `select version, status, executed_at from dbmigrator_version order by executed_at`
)

func (p *Postgres) Connect(dsn string) error {
	p.writeLog("connecting: " + dsn)
	d, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open connect: %w", err)
	}
	if err := p.ConnectExternal(d); err != nil {
		return fmt.Errorf("connect external: %w", err)
	}
	return nil
}

func (p *Postgres) ConnectExternal(d *sqlx.DB) error {
	if err := d.PingContext(p.ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	p.db = d
	return nil
}

func (p *Postgres) Close() error {
	if err := p.db.Close(); err != nil {
		return fmt.Errorf("close db: %w", err)
	}
	return nil
}

func (p *Postgres) FindNewMigrations(ids []int) ([]int, error) {
	query, args, err := sqlx.In(SQLSelectNewMigrations, ids, Processing)
	if err != nil {
		return nil, fmt.Errorf("sqlx in: %w", err)
	}
	query = p.db.Rebind(query)
	rows, err := p.db.QueryxContext(p.ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select new migrations: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	dbIds := []int{}
	for rows.Next() {
		version := 0
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("migrations scan: %w", err)
		}
		dbIds = append(dbIds, version)
	}
	return dbIds, nil
}

func (p *Postgres) CreateMigratorTable() error {
	p.writeLog("executing: " + SQLCreateTable)
	if _, err := p.db.ExecContext(p.ctx, SQLCreateTable); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	p.writeLog("executing: " + SQLCreateTableIndex)
	if _, err := p.db.ExecContext(p.ctx, SQLCreateTableIndex); err != nil {
		return err
	}

	p.writeLog("executing: " + SQLCreateTableComments)
	if _, err := p.db.ExecContext(p.ctx, SQLCreateTableComments); err != nil {
		return fmt.Errorf("comment on: %w", err)
	}
	return nil
}

func (p *Postgres) SelectVersionRow(id int) (VersionRow, error) {
	tx, err := p.db.BeginTxx(p.ctx, nil)
	if err != nil {
		return nil, err
	}
	return &versionRow{id: id, tx: tx}, nil
}

func (p *Postgres) FindLastVersion() (int, error) {
	rows, err := p.db.QueryContext(p.ctx, SQLSelectLastVersion)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	var version int
	if rows.Next() {
		if err := rows.Scan(&version); err != nil {
			return 0, err
		}
	}
	return version, nil
}

func (p *Postgres) DeleteByVersion(version int) error {
	if _, err := p.db.NamedExecContext(p.ctx, SQLDeleteByID, map[string]interface{}{"version": version}); err != nil {
		return err
	}
	return nil
}

func (p *Postgres) FindVersionStatusByVersion(version int) (Status, error) {
	rows, err := p.db.NamedQueryContext(p.ctx, SQLSelectStatusByID, map[string]interface{}{"version": version})
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = rows.Close()
	}()
	var status Status
	if rows.Next() {
		if err := rows.Scan(&status); err != nil {
			return 0, err
		}
	}
	return status, nil
}

func (p *Postgres) SelectRows() ([]Row, error) {
	rows, err := p.db.QueryContext(p.ctx, SQLSelectAll)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
		_ = rows.Err()
	}()
	res := []Row{}
	for rows.Next() {
		r := Row{}
		if err := rows.Scan(&r.Version, &r.Status, &r.ExecutedAt); err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (p *Postgres) writeLog(s string) {
	_, _ = p.log.Write([]byte(s + "\n"))
}

type versionRow struct {
	id int
	tx *sqlx.Tx
}

func (v versionRow) CommitSuccess() error {
	if err := v.SetStatus(Success); err != nil {
		return fmt.Errorf("status success: %w", err)
	}
	if err := v.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (v versionRow) CommitError() error {
	if err := v.SetStatus(Error); err != nil {
		return fmt.Errorf("status error: %w", err)
	}
	if err := v.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (p *Postgres) ExecSQL(sql string) error {
	if _, err := p.db.Exec(sql); err != nil {
		return err
	}
	return nil
}

func (p *Postgres) GetVersionsByStatus(status Status) ([]int, error) {
	rows, err := p.db.NamedQueryContext(p.ctx, SQLSelectByStatus, map[string]interface{}{"status": status})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	versions := []int{}
	for rows.Next() {
		version := 0
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, nil
}

func (v versionRow) Insert() error {
	_, err := v.tx.NamedExec(SQLInsert, map[string]interface{}{"id": v.id, "status": Processing})
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	_, err = v.tx.NamedExec(SQLSelectLock, map[string]interface{}{"id": v.id})
	if err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	return nil
}

func (v versionRow) SetStatus(status Status) error {
	if _, err := v.tx.NamedExec(SQLUpdateStatus, map[string]interface{}{"id": v.id, "status": status}); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func (v versionRow) Commit() error {
	if err := v.tx.Commit(); err != nil {
		return fmt.Errorf("commit version row: %w", err)
	}
	return nil
}

func (v versionRow) Close() error {
	if err := v.tx.Rollback(); err != nil {
		return fmt.Errorf("close version row: %w", err)
	}
	return nil
}

var (
	_ DataKeeper = (*Postgres)(nil)
	_ VersionRow = (*versionRow)(nil)
)
