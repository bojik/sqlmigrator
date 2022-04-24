package migrator

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/bojik/sqlmigrator/pkg/db"
	"github.com/bojik/sqlmigrator/pkg/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

const testDataDir = "./testdata"

func TestMigrator_CreateMigrationSql(t *testing.T) {
	migrator := New(newEmptyLogger())
	wg := sync.WaitGroup{}
	files := []string{}
	mu := sync.Mutex{}
	total := 10
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			up, down, err := migrator.CreateSQLMigration(testDataDir, "init data")
			require.Nil(t, err)
			require.FileExists(t, up)
			require.FileExists(t, down)
			mu.Lock()
			files = append(files, up)
			files = append(files, down)
			mu.Unlock()
		}()
	}
	wg.Wait()
	require.Len(t, files, 2*total)
	for _, file := range files {
		err := os.Remove(file)
		require.Nil(t, err)
	}
}

func TestMigrator_CreateMigrationSqlError(t *testing.T) {
	migrator := New(newEmptyLogger())
	_, _, err := migrator.CreateSQLMigration("./migrator_test.go", "")
	require.ErrorIs(t, err, ErrIsNotDir)
	_, _, err = migrator.CreateSQLMigration("./migrator_test", "")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestGenerateSqlFilenames(t *testing.T) {
	up, down, err := generateSQLFilenames(testDataDir, "suffix тут русский текст suffix")
	require.Nil(t, err)
	require.Regexp(t, regexp.MustCompile(`\d+.suffix____suffix.up.sql$`), up)
	require.Regexp(t, regexp.MustCompile(`\d+.suffix____suffix.down.sql$`), down)
	s := strings.ReplaceAll(down, ".down.", ".up.")
	require.Equal(t, up, s)
}

func TestMigrationUp(t *testing.T) {
	mc := gomock.NewController(t)
	dk := mock.NewMockDataKeeper(mc)
	vr := mock.NewMockVersionRow(mc)
	migrator := New(newEmptyLogger())
	dir := testDataDir
	up, down, err := migrator.CreateSQLMigration(dir, "")
	require.Nil(t, err)
	err1 := ioutil.WriteFile(up, []byte("create table test(id serial primary key, test varchar);"), 0o600)
	require.Nil(t, err1)
	defer func() {
		_ = os.Remove(up)
	}()
	err2 := ioutil.WriteFile(down, []byte("drop table test;"), 0o600)
	require.Nil(t, err2)
	defer func() {
		_ = os.Remove(down)
	}()
	tasks, err4 := migrator.prepareFileTasks([]string{path.Base(up)}, dir)
	require.Nil(t, err4)
	dk.EXPECT().CreateMigratorTable().Return(nil)
	dk.EXPECT().GetVersionsByStatus(db.Error).Return([]int{}, nil)
	dk.EXPECT().FindNewMigrations([]int{tasks[0].id}).Return([]int{}, nil)
	dk.EXPECT().SelectVersionRow(tasks[0].id).Return(vr, nil)
	vr.EXPECT().Insert().Return(nil)
	dk.EXPECT().ExecSql(`create table test(id serial primary key, test varchar);`).Return(nil)
	vr.EXPECT().CommitSuccess().Return(nil)
	vr.EXPECT().Close()
	err3 := migrator.migrateUp(dk, dir)
	require.Nil(t, err3)
}

func TestMigrationDown(t *testing.T) {
	mc := gomock.NewController(t)
	dk := mock.NewMockDataKeeper(mc)
	migrator := New(newEmptyLogger())
	dir := testDataDir
	up, down, err := migrator.CreateSQLMigration(dir, "")
	require.Nil(t, err)
	err1 := ioutil.WriteFile(up, []byte("create table test(id serial primary key, test varchar);"), 0o600)
	require.Nil(t, err1)
	defer func() {
		_ = os.Remove(up)
	}()
	err2 := ioutil.WriteFile(down, []byte("drop table test;"), 0o600)
	require.Nil(t, err2)
	defer func() {
		_ = os.Remove(down)
	}()
	id := getIDByFilename(path.Base(down))
	dk.EXPECT().FindLastVersion().Return(id, nil)
	dk.EXPECT().FindVersionStatusById(id).Return(db.Success, nil)
	dk.EXPECT().ExecSql("drop table test;").Return(nil)
	dk.EXPECT().DeleteById(id).Return(nil)
	err3 := migrator.migrateDown(dk, dir)
	require.Nil(t, err3)
}

func TestMigrationDownWithError(t *testing.T) {
	mc := gomock.NewController(t)
	dk := mock.NewMockDataKeeper(mc)
	migrator := New(newEmptyLogger())
	dir := testDataDir
	up, down, err := migrator.CreateSQLMigration(dir, "")
	require.Nil(t, err)
	err1 := ioutil.WriteFile(up, []byte("create table test(id serial primary key, test varchar);"), 0o600)
	require.Nil(t, err1)
	defer func() {
		_ = os.Remove(up)
	}()
	err2 := ioutil.WriteFile(down, []byte("drop table test;"), 0o600)
	require.Nil(t, err2)
	defer func() {
		_ = os.Remove(down)
	}()
	id := getIDByFilename(path.Base(down))
	dk.EXPECT().FindLastVersion().Return(id, nil)
	dk.EXPECT().FindVersionStatusById(id).Return(db.Error, nil)
	dk.EXPECT().DeleteById(id).Return(nil)
	err3 := migrator.migrateDown(dk, dir)
	require.Nil(t, err3)
}

func TestMigrationDownNotFoundRow(t *testing.T) {
	mc := gomock.NewController(t)
	dk := mock.NewMockDataKeeper(mc)
	migrator := New(newEmptyLogger())
	dir := testDataDir
	up, down, err := migrator.CreateSQLMigration(dir, "")
	require.Nil(t, err)
	err1 := ioutil.WriteFile(up, []byte("create table test(id serial primary key, test varchar);"), 0o600)
	require.Nil(t, err1)
	defer func() {
		_ = os.Remove(up)
	}()
	err2 := ioutil.WriteFile(down, []byte("drop table test;"), 0o600)
	require.Nil(t, err2)
	defer func() {
		_ = os.Remove(down)
	}()
	id := getIDByFilename(path.Base(down))
	dk.EXPECT().FindLastVersion().Return(id, nil)
	dk.EXPECT().FindVersionStatusById(id).Return(db.Status(0), nil)
	err3 := migrator.migrateDown(dk, dir)
	require.ErrorIs(t, err3, ErrVersionHasNotBeenFound)
}

func TestMigrationUpError(t *testing.T) {
	mc := gomock.NewController(t)
	dk := mock.NewMockDataKeeper(mc)
	vr := mock.NewMockVersionRow(mc)
	migrator := New(newEmptyLogger())
	dir := testDataDir
	up, down, err := migrator.CreateSQLMigration(dir, "")
	require.Nil(t, err)
	err1 := ioutil.WriteFile(up, []byte("create table test(id serial primary key, test varchar);"), 0o600)
	require.Nil(t, err1)
	defer func() {
		_ = os.Remove(up)
	}()
	err2 := ioutil.WriteFile(down, []byte("drop table test;"), 0o600)
	require.Nil(t, err2)
	defer func() {
		_ = os.Remove(down)
	}()
	tasks, err4 := migrator.prepareFileTasks([]string{path.Base(up)}, dir)
	require.Nil(t, err4)
	dk.EXPECT().CreateMigratorTable().Return(nil)
	dk.EXPECT().GetVersionsByStatus(db.Error).Return([]int{}, nil)
	dk.EXPECT().FindNewMigrations([]int{tasks[0].id}).Return([]int{}, nil)
	dk.EXPECT().SelectVersionRow(tasks[0].id).Return(vr, nil)
	vr.EXPECT().Insert().Return(nil)
	sqlErr := errors.New("some sql error")
	dk.EXPECT().ExecSql(`create table test(id serial primary key, test varchar);`).Return(sqlErr)
	vr.EXPECT().CommitError().Return(nil)
	vr.EXPECT().Close()
	err3 := migrator.migrateUp(dk, dir)
	//nolint:errorlint
	mErr, ok := err3.(MigrationError)
	require.True(t, ok)
	require.ErrorIs(t, mErr.Err, sqlErr)
	require.Equal(t, mErr.File, up)
	require.Equal(t, mErr.SQL, `create table test(id serial primary key, test varchar);`)
}

func newEmptyLogger() *emptyLogger {
	return &emptyLogger{}
}

type emptyLogger struct{}

func (*emptyLogger) Write([]byte) (n int, err error) {
	return 0, nil
}

var _ io.Writer = (*emptyLogger)(nil)
