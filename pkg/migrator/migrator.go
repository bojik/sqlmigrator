package migrator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bojik/sqlmigrator/pkg/db"
	"github.com/jmoiron/sqlx"
	// postgresql driver.
	_ "github.com/lib/pq"
)

const FilePermission = 0o644

const (
	SQLFileExtension     = ".sql"
	SQLUpFileExtension   = ".up" + SQLFileExtension
	SQLDownFileExtension = ".down" + SQLFileExtension
)

type Type int

const (
	Up Type = iota + 1
	Down
)

type Migrator struct {
	logWriter io.Writer
	mu        sync.Mutex
}

type Result struct {
	Type
	File    string
	SQL     string
	Version int
	db.Status
	Err error
}

type ResultArchive struct {
	Version    int
	Status     db.Status
	ExecutedAt *time.Time
}

type sqlTask struct {
	version  int
	file     string
	sql      string
	finished bool
}

func New(logWriter io.Writer) *Migrator {
	return &Migrator{
		logWriter: logWriter,
		mu:        sync.Mutex{},
	}
}

func (m *Migrator) CreateSQLMigration(dir, suffix string) (string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := validateDir(dir); err != nil {
		return "", "", err
	}
	up, down, err := generateSQLFilenames(dir, suffix)
	if err != nil {
		return "", "", err
	}
	if err := ioutil.WriteFile(up, []byte(""), FilePermission); err != nil {
		return "", "", err
	}
	if err := ioutil.WriteFile(down, []byte(""), FilePermission); err != nil {
		return "", "", err
	}
	return up, down, nil
}

func (m *Migrator) ApplyUpSQLMigrationConnection(ctx context.Context, conn *sqlx.DB, dir string) ([]*Result, error) {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.ConnectExternal(conn); err != nil {
		return nil, fmt.Errorf("connect external: %w", err)
	}
	return m.migrateUp(d, dir)
}

func (m *Migrator) ApplyUpSQLMigration(ctx context.Context, dsn, dir string) ([]*Result, error) {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.Connect(dsn); err != nil {
		return nil, fmt.Errorf("connect external: %w", err)
	}
	return m.migrateUp(d, dir)
}

func (m *Migrator) ApplyDownSQLMigrationConnection(ctx context.Context, conn *sqlx.DB, dir string) ([]*Result, error) {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.ConnectExternal(conn); err != nil {
		return nil, fmt.Errorf("connect external: %w", err)
	}
	return m.migrateDown(d, dir)
}

func (m *Migrator) ApplyDownSQLMigration(ctx context.Context, dsn, dir string) ([]*Result, error) {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.Connect(dsn); err != nil {
		return nil, fmt.Errorf("connect external: %w", err)
	}
	return m.migrateDown(d, dir)
}

func (m *Migrator) ApplyRedoSQLMigrationConnection(ctx context.Context, conn *sqlx.DB, dir string) ([]*Result, error) {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.ConnectExternal(conn); err != nil {
		return nil, fmt.Errorf("connect external: %w", err)
	}
	return m.migrateRedo(d, dir)
}

func (m *Migrator) ApplyRedoSQLMigration(ctx context.Context, dsn, dir string) ([]*Result, error) {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.Connect(dsn); err != nil {
		return nil, fmt.Errorf("connect external: %w", err)
	}
	return m.migrateRedo(d, dir)
}

func (m *Migrator) SelectStatusesConnection(ctx context.Context, conn *sqlx.DB) ([]*ResultArchive, error) {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.ConnectExternal(conn); err != nil {
		return nil, fmt.Errorf("connect external: %w", err)
	}
	return m.selectStatuses(d)
}

func (m *Migrator) SelectStatuses(ctx context.Context, dsn string) ([]*ResultArchive, error) {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.Connect(dsn); err != nil {
		return nil, fmt.Errorf("connect external: %w", err)
	}
	return m.selectStatuses(d)
}

func (m *Migrator) SelectDBVersionConnection(ctx context.Context, conn *sqlx.DB) (int, error) {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.ConnectExternal(conn); err != nil {
		return 0, fmt.Errorf("connect external: %w", err)
	}
	return m.selectDBVersion(d)
}

func (m *Migrator) SelectDBVersion(ctx context.Context, dsn string) (int, error) {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.Connect(dsn); err != nil {
		return 0, fmt.Errorf("connect external: %w", err)
	}
	return m.selectDBVersion(d)
}

func (m *Migrator) selectDBVersion(d db.DataKeeper) (int, error) {
	version, err := d.FindLastVersion()
	if err != nil {
		return 0, err
	}
	return version, nil
}

func (m *Migrator) selectStatuses(d db.DataKeeper) ([]*ResultArchive, error) {
	rows, err := d.SelectRows()
	if err != nil {
		return nil, err
	}
	res := []*ResultArchive{}
	for _, row := range rows {
		res = append(res, &ResultArchive{Version: row.Version, ExecutedAt: row.ExecutedAt, Status: row.Status})
	}
	return res, nil
}

func (m *Migrator) migrateUp(d db.DataKeeper, dir string) ([]*Result, error) {
	if err := d.CreateMigratorTable(); err != nil {
		return nil, fmt.Errorf("create migrator table: %w", err)
	}

	versions, err := d.GetVersionsByStatus(db.Error)
	if err != nil {
		return nil, err
	}
	if len(versions) > 0 {
		return nil, fmt.Errorf("%w: %d", ErrUnfinishedMigrations, versions[0])
	}

	files, err1 := findFiles(dir, SQLUpFileExtension)
	if err1 != nil {
		return nil, fmt.Errorf("find files: %w", err1)
	}

	tasks, err2 := m.prepareFileTasks(files, dir)
	if err2 != nil {
		return nil, fmt.Errorf("prepare tasks: %w", err2)
	}

	results, err3 := m.executeTasks(d, tasks)
	if err3 != nil {
		return nil, fmt.Errorf("execute tasks: %w", err3)
	}
	return results, nil
}

func (m *Migrator) migrateUpVersion(d db.DataKeeper, dir string, version int) ([]*Result, error) {
	if err := d.CreateMigratorTable(); err != nil {
		return nil, fmt.Errorf("create migrator table: %w", err)
	}

	versions, err := d.GetVersionsByStatus(db.Error)
	if err != nil {
		return nil, err
	}
	if len(versions) > 0 {
		return nil, fmt.Errorf("%w: %d", ErrUnfinishedMigrations, versions[0])
	}

	files, err1 := findFiles(dir, SQLUpFileExtension)
	if err1 != nil {
		return nil, fmt.Errorf("find files: %w", err1)
	}

	tasks, err2 := m.prepareFileTasks(files, dir)
	if err2 != nil {
		return nil, fmt.Errorf("prepare tasks: %w", err2)
	}

	filteredTasks := []*sqlTask{}
	for _, t := range tasks {
		if t.version == version {
			filteredTasks = append(filteredTasks, t)
			break
		}
	}

	if len(filteredTasks) == 0 {
		return nil, fmt.Errorf("%w: %d", ErrVersionHasNotBeenFound, version)
	}

	results, err3 := m.executeTasks(d, filteredTasks)
	if err3 != nil {
		return nil, fmt.Errorf("execute tasks: %w", err3)
	}
	return results, nil
}

func (m *Migrator) migrateDown(d db.DataKeeper, dir string) ([]*Result, error) {
	if err := d.CreateMigratorTable(); err != nil {
		return nil, fmt.Errorf("create migrator table: %w", err)
	}
	version, err := d.FindLastVersion()
	if err != nil {
		return nil, err
	}
	if version > 0 {
		return m.migrateDownVersion(d, dir, version)
	}
	return nil, nil
}

func (m *Migrator) migrateDownVersion(d db.DataKeeper, dir string, version int) ([]*Result, error) {
	files, err := findFiles(dir, SQLDownFileExtension)
	if err != nil {
		return nil, err
	}
	status, err := d.FindVersionStatusByVersion(version)
	if err != nil {
		return nil, err
	}
	if status == 0 {
		return nil, fmt.Errorf("%w: %d", ErrVersionHasNotBeenFound, version)
	}
	results := []*Result{}
	for _, file := range files {
		if getVersionByFilename(file) != version {
			continue
		}
		result := &Result{
			Type:    Down,
			Version: version,
			Status:  db.Success,
			File:    path.Join(dir, file),
		}
		sql, err := ioutil.ReadFile(result.File)
		if err != nil {
			return nil, err
		}
		if status == db.Success {
			result.SQL = string(sql)
			err := d.ExecSQL(result.SQL)
			if err != nil {
				result.Err = err
				result.Status = db.Error
			}
		}
		if err := d.DeleteByVersion(version); err != nil {
			return nil, err
		}
		results = append(results, result)
		break
	}
	return results, nil
}

func (m *Migrator) migrateRedo(d db.DataKeeper, dir string) ([]*Result, error) {
	result, err := m.migrateDown(d, dir)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	resultUp, err1 := m.migrateUpVersion(d, dir, result[len(result)-1].Version)
	if err1 != nil {
		return nil, err1
	}
	result = append(result, resultUp...)
	return result, nil
}

func (m *Migrator) executeTasks(d db.DataKeeper, tasks []*sqlTask) ([]*Result, error) {
	ids := make([]int, len(tasks))
	taskMap := make(map[int]*sqlTask, len(tasks))
	for i, task := range tasks {
		ids[i] = task.version
		taskMap[task.version] = task
	}
	dbIds, err := d.FindNewMigrations(ids)
	if err != nil {
		return nil, fmt.Errorf("find new migrations: %w", err)
	}
	for _, id := range dbIds {
		taskMap[id].finished = true
	}
	results := []*Result{}
	for _, task := range tasks {
		if task.finished {
			continue
		}
		if task.file != "" && task.sql == "" {
			sql, err := ioutil.ReadFile(task.file)
			if err != nil {
				return nil, fmt.Errorf("file reading error: %w", err)
			}
			if len(sql) == 0 {
				m.writeLog("file is empty: " + task.file)
				continue
			}
			task.sql = string(sql)
		}
		result := &Result{
			Type:    Up,
			File:    task.file,
			SQL:     task.sql,
			Version: task.version,
		}
		if err := m.executeTask(d, task); err != nil {
			//nolint:errorlint
			if err2, ok := err.(migrationError); ok {
				result.Status = db.Error
				result.Err = err2.Err
				results = append(results, result)
				return results, nil
			}
			return nil, fmt.Errorf("execute task: %w", err)
		}
		result.Status = db.Success
		results = append(results, result)
	}
	return results, nil
}

func (m *Migrator) executeTask(d db.DataKeeper, task *sqlTask) error {
	version, err := d.SelectVersionRow(task.version)
	if err != nil {
		return fmt.Errorf("select verion row: %w", err)
	}
	defer func() {
		_ = version.Close()
	}()
	if err := version.Insert(); err != nil {
		return fmt.Errorf("insert version row: %w", err)
	}
	if err := d.ExecSQL(task.sql); err != nil {
		e := newMigrationError(err, task.sql, task.file)
		if err := version.CommitError(); err != nil {
			return fmt.Errorf("commit success: %w", err)
		}
		return e
	}
	if err := version.CommitSuccess(); err != nil {
		return fmt.Errorf("commit success: %w", err)
	}
	return nil
}

func (m *Migrator) prepareFileTasks(files []string, dir string) ([]*sqlTask, error) {
	tasks := []*sqlTask{}
	for _, file := range files {
		downFile := strings.ReplaceAll(file, SQLUpFileExtension, SQLDownFileExtension)
		exists, err := isFileExist(path.Join(dir, downFile))
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("%w: for %s", ErrDownMigrationIsNotExist, path.Join(dir, file))
		}
		task := &sqlTask{
			version: getVersionByFilename(file),
			file:    path.Join(dir, file),
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func generateSQLFilenames(dir, suffix string) (string, string, error) {
	for {
		prefix := fmt.Sprintf(
			"%s%d",
			time.Now().Format("20060102150405"),
			time.Now().Nanosecond()/int(time.Millisecond),
		)
		if suffix != "" {
			prefix = fmt.Sprintf("%s.%s", prefix, prepareSuffix(suffix))
		}
		up := path.Join(dir, prefix+SQLUpFileExtension)
		down := path.Join(dir, prefix+SQLDownFileExtension)
		upExists, err := isFileExist(up)
		if err != nil {
			return "", "", err
		}
		downExists, err := isFileExist(down)
		if err != nil {
			return "", "", err
		}
		if !upExists && !downExists {
			return up, down, nil
		}
		time.Sleep(time.Millisecond)
	}
}

func prepareSuffix(suffix string) string {
	str := strings.ReplaceAll(suffix, " ", "_")
	re := regexp.MustCompile("[^a-zA-Z0-9_]+")
	ret := re.ReplaceAll([]byte(str), []byte(""))
	return string(ret)
}

func isFileExist(fullPath string) (bool, error) {
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func validateDir(dir string) error {
	dirStat, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !dirStat.IsDir() {
		return fmt.Errorf("%w: %s", ErrIsNotDir, dir)
	}
	return nil
}

func (m *Migrator) writeLog(s string) {
	_, _ = m.logWriter.Write([]byte(s + "\n"))
}

func findFiles(dir, suffix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name[len(name)-len(suffix):] != suffix {
			continue
		}
		files = append(files, name)
	}
	return files, nil
}

func getVersionByFilename(file string) int {
	parts := strings.Split(file, ".")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return id
}
