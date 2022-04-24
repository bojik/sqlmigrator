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

type Migrator struct {
	logWriter io.Writer
	mu        sync.Mutex
}

type migratorSQLTask struct {
	id       int
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

func (m *Migrator) CreateSQLMigration(migrationDir, suffix string) (string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := validateDir(migrationDir); err != nil {
		return "", "", err
	}
	up, down, err := generateSQLFilenames(migrationDir, suffix)
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

func (m *Migrator) ApplyUpSQLMigrationConnection(ctx context.Context, conn *sqlx.DB, migrationDir string) error {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.ConnectExternal(conn); err != nil {
		return fmt.Errorf("connect external: %w", err)
	}
	return m.migrateUp(d, migrationDir)
}

func (m *Migrator) ApplyUpSQLMigration(ctx context.Context, dsn, migrationDir string) error {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.Connect(dsn); err != nil {
		return fmt.Errorf("connect external: %w", err)
	}
	return m.migrateUp(d, migrationDir)
}

func (m *Migrator) ApplyDownSQLMigrationConnection(ctx context.Context, conn *sqlx.DB, migrationDir string) error {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.ConnectExternal(conn); err != nil {
		return fmt.Errorf("connect external: %w", err)
	}
	return m.migrateDown(d, migrationDir)
}

func (m *Migrator) ApplyDownSQLMigration(ctx context.Context, dsn, migrationDir string) error {
	d := db.NewPostgres(ctx, m.logWriter)
	if err := d.Connect(dsn); err != nil {
		return fmt.Errorf("connect external: %w", err)
	}
	return m.migrateDown(d, migrationDir)
}

func (m *Migrator) migrateUp(d db.DataKeeper, migrationDir string) error {
	if err := d.CreateMigratorTable(); err != nil {
		return fmt.Errorf("create migrator table: %w", err)
	}

	versions, err := d.GetVersionsByStatus(db.Error)
	if err != nil {
		return err
	}
	if len(versions) > 0 {
		return fmt.Errorf("%w: %d", ErrUnfinishedMigrations, versions[0])
	}

	files, err1 := findFiles(migrationDir, SQLUpFileExtension)
	if err1 != nil {
		return fmt.Errorf("find files: %w", err1)
	}

	tasks, err2 := m.prepareFileTasks(files, migrationDir)
	if err2 != nil {
		return fmt.Errorf("prepare tasks: %w", err2)
	}

	if err := m.executeTasks(d, tasks); err != nil {
		//nolint:errorlint
		if _, ok := err.(MigrationError); ok {
			return err
		}
		return fmt.Errorf("execute tasks: %w", err)
	}
	return nil
}

func (m *Migrator) migrateDown(d db.DataKeeper, migrationDir string) error {
	version, err := d.FindLastVersion()
	if err != nil {
		return err
	}
	if version > 0 {
		return m.migrateDownVersion(d, migrationDir, version)
	}
	return nil
}

func (m *Migrator) migrateDownVersion(d db.DataKeeper, migrationDir string, version int) error {
	files, err := findFiles(migrationDir, SQLDownFileExtension)
	if err != nil {
		return err
	}
	status, err := d.FindVersionStatusByID(version)
	if err != nil {
		return err
	}
	if status == 0 {
		return fmt.Errorf("%w: %d", ErrVersionHasNotBeenFound, version)
	}
	for _, file := range files {
		if getIDByFilename(file) == version {
			sql, err := ioutil.ReadFile(path.Join(migrationDir, file))
			if err != nil {
				return err
			}
			if status == db.Success {
				_ = d.ExecSQL(string(sql))
			}
			if err := d.DeleteByID(version); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (m *Migrator) executeTasks(d db.DataKeeper, tasks []*migratorSQLTask) error {
	ids := make([]int, len(tasks))
	taskMap := make(map[int]*migratorSQLTask, len(tasks))
	for i, task := range tasks {
		ids[i] = task.id
		taskMap[task.id] = task
	}
	dbIds, err := d.FindNewMigrations(ids)
	if err != nil {
		return fmt.Errorf("find new migrations: %w", err)
	}
	for _, id := range dbIds {
		taskMap[id].finished = true
	}
	for _, task := range tasks {
		if task.finished {
			continue
		}
		if task.file != "" && task.sql == "" {
			sql, err := ioutil.ReadFile(task.file)
			if err != nil {
				return fmt.Errorf("file reading error: %w", err)
			}
			if len(sql) == 0 {
				m.writeLog("file is empty: " + task.file)
				continue
			}
			task.sql = string(sql)
		}
		if err := m.executeTask(d, task); err != nil {
			//nolint:errorlint
			if _, ok := err.(MigrationError); ok {
				return err
			}
			return fmt.Errorf("execute task: %w", err)
		}
	}
	return nil
}

func (m *Migrator) executeTask(d db.DataKeeper, task *migratorSQLTask) error {
	version, err := d.SelectVersionRow(task.id)
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
		e := NewMigrationError(err, task.sql, task.file)
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

func (m *Migrator) prepareFileTasks(files []string, migrationDir string) ([]*migratorSQLTask, error) {
	tasks := []*migratorSQLTask{}
	for _, file := range files {
		downFile := strings.ReplaceAll(file, SQLUpFileExtension, SQLDownFileExtension)
		exists, err := isFileExist(path.Join(migrationDir, downFile))
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("%w: for %s", ErrDownMigrationIsNotExist, path.Join(migrationDir, file))
		}
		task := &migratorSQLTask{
			id:   getIDByFilename(file),
			file: path.Join(migrationDir, file),
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

func getIDByFilename(file string) int {
	parts := strings.Split(file, ".")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return id
}
