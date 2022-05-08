package migrator

import (
	"bytes"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/bojik/sqlmigrator/pkg/db"
	"github.com/bojik/sqlmigrator/pkg/mock"
	"github.com/golang/mock/gomock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestMigrator_CreateMigrationGo(t *testing.T) {
	migrator := New(newEmptyLogger())
	wg := sync.WaitGroup{}
	files := []string{}
	mu := sync.Mutex{}
	total := 10
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			file, err := migrator.CreateGoMigration(testDataDir, "init data")
			require.Nil(t, err)
			require.FileExists(t, file)
			b, err := ioutil.ReadFile(file)
			require.Nil(t, err)
			buff := &bytes.Buffer{}
			err1 := goTemplate.Execute(buff, struct{}{})
			require.Nil(t, err1)
			require.Equal(t, string(b), buff.String())
			mu.Lock()
			files = append(files, file)
			mu.Unlock()
		}()
	}
	wg.Wait()
	require.Len(t, files, total)
	for _, file := range files {
		err := os.Remove(file)
		require.Nil(t, err)
	}
}

func TestMigrator_ApplyGoMigration(t *testing.T) {
	migrator := New(newEmptyLogger())
	Exec.Add(func(db *sqlx.DB) (string, error) {
		return "select 1", nil
	}, func(db *sqlx.DB) (string, error) {
		return "select 2", nil
	})
	mc := gomock.NewController(t)
	dk := mock.NewMockDataKeeper(mc)
	vr := mock.NewMockVersionRow(mc)
	sqlxdb := &sqlx.DB{}
	dk.EXPECT().CreateMigratorTable().Return(nil)
	dk.EXPECT().GetVersionsByStatus(db.Error).Return([]int{}, nil)
	dk.EXPECT().GetDB().Return(sqlxdb)
	dk.EXPECT().FindNewMigrations([]int{20220507}).Return([]int{}, nil)
	dk.EXPECT().SelectVersionRow(20220507).Return(vr, nil)
	vr.EXPECT().Insert().Return(nil)
	dk.EXPECT().ExecSQL(`select 1`).Return(nil)
	vr.EXPECT().CommitSuccess().Return(nil)
	vr.EXPECT().Close()
	res, err := migrator.migrateUpGo(dk)
	require.Nil(t, err)
	require.Len(t, res, 1)
	require.Equal(t, "select 1", res[0].SQL)

	dk.EXPECT().CreateMigratorTable().Return(nil)
	dk.EXPECT().FindLastVersion().Return(20220507, nil)
	dk.EXPECT().FindVersionStatusByVersion(20220507).Return(db.Success, nil)
	dk.EXPECT().GetDB().Return(sqlxdb)
	dk.EXPECT().ExecSQL(`select 2`).Return(nil)
	dk.EXPECT().DeleteByVersion(20220507).Return(nil)
	dk.EXPECT().CreateMigratorTable().Return(nil)
	dk.EXPECT().GetVersionsByStatus(db.Error).Return([]int{}, nil)
	dk.EXPECT().GetDB().Return(sqlxdb)
	dk.EXPECT().FindNewMigrations([]int{20220507}).Return([]int{}, nil)
	dk.EXPECT().SelectVersionRow(20220507).Return(vr, nil)
	vr.EXPECT().Insert().Return(nil)
	dk.EXPECT().ExecSQL(`select 1`).Return(nil)
	vr.EXPECT().CommitSuccess().Return(nil)
	vr.EXPECT().Close()
	res, err = migrator.migrateRedoGo(dk)
	require.Nil(t, err)
	require.Len(t, res, 2)
	require.Equal(t, "select 2", res[0].SQL)
	require.Equal(t, Down, res[0].Type)
	require.Equal(t, "select 1", res[1].SQL)
	require.Equal(t, Up, res[1].Type)
	Exec.reset()
}
