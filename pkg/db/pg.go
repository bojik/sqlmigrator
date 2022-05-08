package db

import (
	"context"
	"fmt"
	"io"

	"github.com/bojik/sqlmigrator/pkg/config"
	"github.com/jmoiron/sqlx"
	// postgresql driver.
	_ "github.com/lib/pq"
)

type Postgres struct {
	db    *sqlx.DB
	ctx   context.Context
	log   io.Writer
	table sqlTable
}

func NewPostgres(ctx context.Context, log io.Writer) *Postgres {
	return &Postgres{
		ctx:   ctx,
		log:   log,
		table: sqlTable(config.GetTable()),
	}
}

//nolint:lll
var (
	sqlCreateTable         = sqlTemplate(`create table if not exists {{table}} (version bigint not null primary key, status smallint not null, executed_at timestamp with time zone);`)
	sqlCreateTableIndex    = sqlTemplate(`create index if not exists status_idx on {{table}}(status);create index if not exists executed_at_idx on {{table}}(executed_at);`)
	sqlCreateTableComments = sqlTemplate(`comment on table {{table}} is 'dbmigrator migrations';
			comment on column {{table}}.version is 'migration id';
			comment on column {{table}}.status is '1 - new, 2 - successful finished, 3 - executed with error';
			comment on column {{table}}.executed_at is 'date of the last executing attempt';`)
	sqlSelectNewMigrations = sqlTemplate(`select version from {{table}} where version in (?) and status = ? for update`)
	sqlInsert              = sqlTemplate(`insert into {{table}}(version, status) values(:id, :status) on conflict do nothing`)
	sqlSelectLock          = sqlTemplate(`select version from {{table}} where version = :id for update`)
	sqlUpdateStatus        = sqlTemplate(`update {{table}} set status = :status, executed_at = current_timestamp where version = :id`)
	sqlSelectByStatus      = sqlTemplate(`select version from {{table}} where status = :status order by version`)
	sqlSelectStatusByID    = sqlTemplate(`select status from {{table}} where version = :version`)
	sqlSelectLastVersion   = sqlTemplate(`select version from {{table}} where executed_at = (select max(executed_at) from {{table}})`)
	sqlDeleteByID          = sqlTemplate(`delete from {{table}} where version = :version`)
	sqlSelectAll           = sqlTemplate(`select version, status, COALESCE(executed_at, now()) from {{table}} order by executed_at`)
)

func (p *Postgres) Connect(dsn string) error {
	p.writeLog("connecting: " + dsn)
	d, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open connect: %w", err)
	}
	if err := p.ConnectExternal(d); err != nil {
		return fmt.Errorf("connect: %w", err)
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
	p.writeLog("closing db")
	if err := p.db.Close(); err != nil {
		return fmt.Errorf("close db: %w", err)
	}
	return nil
}

func (p *Postgres) FindNewMigrations(ids []int) ([]int, error) {
	query, args, err := sqlx.In(sqlSelectNewMigrations.sql(p.table), ids, Processing)
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
	p.writeLog("sql: " + sqlCreateTable.sql(p.table))
	if _, err := p.db.ExecContext(p.ctx, sqlCreateTable.sql(p.table)); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	p.writeLog("sql: " + sqlCreateTableIndex.sql(p.table))
	if _, err := p.db.ExecContext(p.ctx, sqlCreateTableIndex.sql(p.table)); err != nil {
		return err
	}

	p.writeLog("sql: " + sqlCreateTableComments.sql(p.table))
	if _, err := p.db.ExecContext(p.ctx, sqlCreateTableComments.sql(p.table)); err != nil {
		return fmt.Errorf("comment on: %w", err)
	}
	return nil
}

func (p *Postgres) SelectVersionRow(id int) (VersionRow, error) {
	tx, err := p.db.BeginTxx(p.ctx, nil)
	if err != nil {
		return nil, err
	}
	return &versionRow{id: id, tx: tx, table: p.table}, nil
}

func (p *Postgres) FindLastVersion() (int, error) {
	p.writeLog("sql: " + sqlSelectLastVersion.sql(p.table))
	rows, err := p.db.QueryContext(p.ctx, sqlSelectLastVersion.sql(p.table))
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
	p.writeLog("sql: " + sqlDeleteByID.sql(p.table))
	_, err := p.db.NamedExecContext(p.ctx, sqlDeleteByID.sql(p.table), map[string]interface{}{"version": version})
	if err != nil {
		return err
	}
	return nil
}

func (p *Postgres) FindVersionStatusByVersion(version int) (Status, error) {
	p.writeLog("sql: " + sqlSelectStatusByID.sql(p.table))
	rows, err := p.db.NamedQueryContext(
		p.ctx,
		sqlSelectStatusByID.sql(p.table),
		map[string]interface{}{"version": version},
	)
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
	p.writeLog("sql: " + sqlSelectAll.sql(p.table))
	rows, err := p.db.QueryContext(p.ctx, sqlSelectAll.sql(p.table))
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
		if r.Status == Processing {
			r.ExecutedAt = nil
		}
		res = append(res, r)
	}
	return res, nil
}

func (p *Postgres) GetDB() *sqlx.DB {
	return p.db
}

func (p *Postgres) writeLog(s string) {
	_, _ = p.log.Write([]byte("> " + s + "\n"))
}

type versionRow struct {
	id       int
	tx       *sqlx.Tx
	finished bool
	table    sqlTable
}

func (v *versionRow) CommitSuccess() error {
	if err := v.SetStatus(Success); err != nil {
		return fmt.Errorf("status success: %w", err)
	}
	if err := v.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (v *versionRow) CommitError() error {
	if err := v.SetStatus(Error); err != nil {
		return fmt.Errorf("status error: %w", err)
	}
	if err := v.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (p *Postgres) ExecSQL(sql string) error {
	p.writeLog("sql: " + sql)
	if _, err := p.db.Exec(sql); err != nil {
		return err
	}
	return nil
}

func (p *Postgres) GetVersionsByStatus(status Status) ([]int, error) {
	p.writeLog("sql: " + sqlSelectByStatus.sql(p.table))
	rows, err := p.db.NamedQueryContext(p.ctx, sqlSelectByStatus.sql(p.table), map[string]interface{}{"status": status})
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

func (v *versionRow) Insert() error {
	_, err := v.tx.NamedExec(sqlInsert.sql(v.table), map[string]interface{}{"id": v.id, "status": Processing})
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	_, err = v.tx.NamedExec(sqlSelectLock.sql(v.table), map[string]interface{}{"id": v.id})
	if err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	return nil
}

func (v *versionRow) SetStatus(status Status) error {
	_, err := v.tx.NamedExec(sqlUpdateStatus.sql(v.table), map[string]interface{}{"id": v.id, "status": status})
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func (v *versionRow) Commit() error {
	if err := v.tx.Commit(); err != nil {
		return fmt.Errorf("commit version row: %w", err)
	}
	v.finished = true
	return nil
}

func (v *versionRow) Close() error {
	if v.finished {
		return nil
	}
	if err := v.tx.Rollback(); err != nil {
		return fmt.Errorf("close version row: %w", err)
	}
	v.finished = true
	return nil
}

var (
	_ DataKeeper = (*Postgres)(nil)
	_ VersionRow = (*versionRow)(nil)
)
