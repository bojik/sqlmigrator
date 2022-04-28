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
	err1 := ioutil.WriteFile(up, []byte("create table test(version serial primary key, test varchar);"), 0o600)
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
	dk.EXPECT().FindNewMigrations([]int{tasks[0].version}).Return([]int{}, nil)
	dk.EXPECT().SelectVersionRow(tasks[0].version).Return(vr, nil)
	vr.EXPECT().Insert().Return(nil)
	dk.EXPECT().ExecSQL(`create table test(version serial primary key, test varchar);`).Return(nil)
	vr.EXPECT().CommitSuccess().Return(nil)
	vr.EXPECT().Close()
	res, err3 := migrator.migrateUp(dk, dir)
	require.Nil(t, err3)
	require.Len(t, res, 1)
	require.Equal(t, `create table test(version serial primary key, test varchar);`, res[0].SQL)
	require.Nil(t, res[0].Err)
	require.Equal(t, db.Success, res[0].Status)
	require.Equal(t, tasks[0].version, res[0].Version)
	require.Equal(t, Up, res[0].Type)
	require.Equal(t, up, res[0].File)
}

func TestMigrationDown(t *testing.T) {
	mc := gomock.NewController(t)
	dk := mock.NewMockDataKeeper(mc)
	migrator := New(newEmptyLogger())
	dir := testDataDir
	up, down, err := migrator.CreateSQLMigration(dir, "")
	require.Nil(t, err)
	err1 := ioutil.WriteFile(up, []byte("create table test(version serial primary key, test varchar);"), 0o600)
	require.Nil(t, err1)
	defer func() {
		_ = os.Remove(up)
	}()
	err2 := ioutil.WriteFile(down, []byte("drop table test;"), 0o600)
	require.Nil(t, err2)
	defer func() {
		_ = os.Remove(down)
	}()
	id := getVersionByFilename(path.Base(down))
	dk.EXPECT().CreateMigratorTable().Return(nil)
	dk.EXPECT().FindLastVersion().Return(id, nil)
	dk.EXPECT().FindVersionStatusByVersion(id).Return(db.Success, nil)
	dk.EXPECT().ExecSQL("drop table test;").Return(nil)
	dk.EXPECT().DeleteByVersion(id).Return(nil)
	res, err3 := migrator.migrateDown(dk, dir)
	require.Nil(t, err3)
	require.Len(t, res, 1)
	require.Equal(t, `drop table test;`, res[0].SQL)
	require.Nil(t, res[0].Err)
	require.Equal(t, db.Success, res[0].Status)
	require.Equal(t, id, res[0].Version)
	require.Equal(t, Down, res[0].Type)
	require.Equal(t, down, res[0].File)
}

func TestMigrationDownWithError(t *testing.T) {
	mc := gomock.NewController(t)
	dk := mock.NewMockDataKeeper(mc)
	migrator := New(newEmptyLogger())
	dir := testDataDir
	up, down, err := migrator.CreateSQLMigration(dir, "")
	require.Nil(t, err)
	err1 := ioutil.WriteFile(up, []byte("create table test(version serial primary key, test varchar);"), 0o600)
	require.Nil(t, err1)
	defer func() {
		_ = os.Remove(up)
	}()
	err2 := ioutil.WriteFile(down, []byte("drop table test;"), 0o600)
	require.Nil(t, err2)
	defer func() {
		_ = os.Remove(down)
	}()
	id := getVersionByFilename(path.Base(down))
	dk.EXPECT().CreateMigratorTable().Return(nil)
	dk.EXPECT().FindLastVersion().Return(id, nil)
	dk.EXPECT().FindVersionStatusByVersion(id).Return(db.Error, nil)
	dk.EXPECT().DeleteByVersion(id).Return(nil)
	_, err3 := migrator.migrateDown(dk, dir)
	require.Nil(t, err3)
}

func TestMigrationDownNotFoundRow(t *testing.T) {
	mc := gomock.NewController(t)
	dk := mock.NewMockDataKeeper(mc)
	migrator := New(newEmptyLogger())
	dir := testDataDir
	up, down, err := migrator.CreateSQLMigration(dir, "")
	require.Nil(t, err)
	err1 := ioutil.WriteFile(up, []byte("create table test(version serial primary key, test varchar);"), 0o600)
	require.Nil(t, err1)
	defer func() {
		_ = os.Remove(up)
	}()
	err2 := ioutil.WriteFile(down, []byte("drop table test;"), 0o600)
	require.Nil(t, err2)
	defer func() {
		_ = os.Remove(down)
	}()
	id := getVersionByFilename(path.Base(down))
	dk.EXPECT().CreateMigratorTable()
	dk.EXPECT().FindLastVersion().Return(id, nil)
	dk.EXPECT().FindVersionStatusByVersion(id).Return(db.Status(0), nil)
	_, err3 := migrator.migrateDown(dk, dir)
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
	err1 := ioutil.WriteFile(up, []byte("create table test(version serial primary key, test varchar);"), 0o600)
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
	dk.EXPECT().FindNewMigrations([]int{tasks[0].version}).Return([]int{}, nil)
	dk.EXPECT().SelectVersionRow(tasks[0].version).Return(vr, nil)
	vr.EXPECT().Insert().Return(nil)
	sqlErr := errors.New("some sql error")
	dk.EXPECT().ExecSQL(`create table test(version serial primary key, test varchar);`).Return(sqlErr)
	vr.EXPECT().CommitError().Return(nil)
	vr.EXPECT().Close()
	res, err3 := migrator.migrateUp(dk, dir)
	require.Nil(t, err3)
	require.Len(t, res, 1)
	require.Equal(t, `create table test(version serial primary key, test varchar);`, res[0].SQL)
	require.Equal(t, sqlErr, res[0].Err)
	require.Equal(t, db.Error, res[0].Status)
	require.Equal(t, tasks[0].version, res[0].Version)
	require.Equal(t, Up, res[0].Type)
}

func TestMigratorRedo(t *testing.T) {
	mc := gomock.NewController(t)
	dk := mock.NewMockDataKeeper(mc)
	vr := mock.NewMockVersionRow(mc)
	migrator := New(newEmptyLogger())
	dir := testDataDir
	up, down, err := migrator.CreateSQLMigration(dir, "")
	require.Nil(t, err)
	err1 := ioutil.WriteFile(up, []byte("create table test(version serial primary key, test varchar);"), 0o600)
	require.Nil(t, err1)
	defer func() {
		_ = os.Remove(up)
	}()
	err2 := ioutil.WriteFile(down, []byte("drop table test;"), 0o600)
	require.Nil(t, err2)
	defer func() {
		_ = os.Remove(down)
	}()
	id := getVersionByFilename(path.Base(down))
	dk.EXPECT().CreateMigratorTable().Return(nil)
	dk.EXPECT().FindLastVersion().Return(id, nil)
	dk.EXPECT().FindVersionStatusByVersion(id).Return(db.Success, nil)
	dk.EXPECT().ExecSQL("drop table test;").Return(nil)
	dk.EXPECT().DeleteByVersion(id).Return(nil)
	dk.EXPECT().CreateMigratorTable().Return(nil)
	dk.EXPECT().GetVersionsByStatus(db.Error).Return([]int{}, nil)
	dk.EXPECT().FindNewMigrations([]int{id}).Return([]int{}, nil)
	dk.EXPECT().SelectVersionRow(id).Return(vr, nil)
	vr.EXPECT().Insert().Return(nil)
	dk.EXPECT().ExecSQL(`create table test(version serial primary key, test varchar);`).Return(nil)
	vr.EXPECT().CommitSuccess().Return(nil)
	vr.EXPECT().Close()
	res, err3 := migrator.migrateRedo(dk, dir)
	require.Nil(t, err3)
	require.Len(t, res, 2)
	require.Equal(t, `drop table test;`, res[0].SQL)
	require.Nil(t, res[0].Err)
	require.Equal(t, db.Success, res[0].Status)
	require.Equal(t, id, res[0].Version)
	require.Equal(t, Down, res[0].Type)
	require.Equal(t, down, res[0].File)

	require.Equal(t, `create table test(version serial primary key, test varchar);`, res[1].SQL)
	require.Nil(t, res[1].Err)
	require.Equal(t, db.Success, res[1].Status)
	require.Equal(t, id, res[1].Version)
	require.Equal(t, Up, res[1].Type)
	require.Equal(t, up, res[1].File)
}

func TestMigrator_SelectDbVersion(t *testing.T) {
	mc := gomock.NewController(t)
	dk := mock.NewMockDataKeeper(mc)
	dk.EXPECT().FindLastVersion().Return(4, nil)
	migrator := New(newEmptyLogger())
	v, err := migrator.selectDBVersion(dk)
	require.Nil(t, err)
	require.Equal(t, 4, v)
}

func newEmptyLogger() *emptyLogger {
	return &emptyLogger{}
}

type emptyLogger struct{}

func (*emptyLogger) Write([]byte) (n int, err error) {
	return 0, nil
}

var _ io.Writer = (*emptyLogger)(nil)
