//go:build integration
// +build integration

package db

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PgTestSuite struct {
	p *Postgres
	suite.Suite
}

func (*PgTestSuite) Write([]byte) (int, error) {
	return 0, nil
}

const Dsn = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

func (s *PgTestSuite) SetupSuite() {
	s.p = NewPostgres(context.Background(), s)
	{
		err := s.p.Connect(Dsn)
		require.Nil(s.T(), err)
	}
	{
		err := s.p.ExecSQL("drop table if exists dbmigrator_version")
		require.Nil(s.T(), err)
	}
	{
		err := s.p.CreateMigratorTable()
		require.Nil(s.T(), err)
	}
}

func (s *PgTestSuite) TestAll() {
	s.testMainLogic()
	s.testLockLogic()
}

func (s *PgTestSuite) testMainLogic() {
	tests := []struct {
		id int
		st Status
	}{
		{1, Success},
		{2, Error},
		{3, Processing},
	}
	for _, tc := range tests {
		tc := tc
		s.T().Run(tc.st.String(), func(t *testing.T) {
			row, err := s.p.SelectVersionRow(tc.id)
			require.Nil(s.T(), err)
			defer func() {
				_ = row.Close()
			}()
			err1 := row.Insert()
			require.Nil(s.T(), err1)
			if tc.st == Success {
				err2 := row.CommitSuccess()
				require.Nil(s.T(), err2)
			}
			if tc.st == Error {
				err2 := row.CommitError()
				require.Nil(s.T(), err2)
			}
			if tc.st == Processing {
				err2 := row.Commit()
				require.Nil(s.T(), err2)
			}
			err3 := row.Close()
			require.Nil(s.T(), err3)
			st, err4 := s.p.FindVersionStatusByVersion(tc.id)
			require.Nil(s.T(), err4)
			require.Equal(s.T(), tc.st, st)
		})
	}
	v, err := s.p.FindLastVersion()
	require.Nil(s.T(), err)
	require.Equal(s.T(), 2, v)
	a, err := s.p.FindNewMigrations([]int{1, 2})
	require.Nil(s.T(), err)
	require.Len(s.T(), a, 2)
	require.Equal(s.T(), 1, a[0])
	require.Equal(s.T(), 2, a[1])
	for _, status := range []Status{Success, Error, Processing} {
		versions, err := s.p.GetVersionsByStatus(status)
		require.Nil(s.T(), err)
		require.Len(s.T(), versions, 1)
	}
	rows, err := s.p.SelectRows()
	require.Nil(s.T(), err)
	require.Len(s.T(), rows, 3)
	for i := range tests {
		tc := tests[i]
		row := rows[i]
		require.Equal(s.T(), tc.id, row.Version)
		require.Equal(s.T(), tc.st, row.Status)
		if row.Status == Processing {
			require.Nil(s.T(), row.ExecutedAt)
		} else {
			require.NotNil(s.T(), row.ExecutedAt)
		}
		err := s.p.DeleteByVersion(tc.id)
		require.Nil(s.T(), err)
	}
	rows, err = s.p.SelectRows()
	require.Nil(s.T(), err)
	require.Len(s.T(), rows, 0)
}

func (s *PgTestSuite) testLockLogic() {
	version := 4
	row, err := s.p.SelectVersionRow(version)
	require.Nil(s.T(), err)
	defer func() {
		_ = row.Close()
	}()
	err1 := row.Insert()
	require.Nil(s.T(), err1)
	err2 := row.Commit()
	require.Nil(s.T(), err2)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		row, err := s.p.SelectVersionRow(version)
		require.Nil(s.T(), err)
		defer func() {
			_ = row.Close()
		}()
		err1 := row.Insert()
		require.Nil(s.T(), err1)
		time.Sleep(time.Second)
		err2 = row.CommitSuccess()
		require.Nil(s.T(), err2)
	}()
	// open new connect for test purity
	p := NewPostgres(context.Background(), s)
	err3 := p.Connect(Dsn)
	require.Nil(s.T(), err3)
	defer func() {
		_ = p.Close()
	}()
	rows, err := p.FindNewMigrations([]int{version})
	require.Nil(s.T(), err)
	require.Len(s.T(), rows, 0)
	wg.Wait()
}

func (s *PgTestSuite) TearDownSuite() {
	err := s.p.Close()
	require.Nil(s.T(), err)
}

func TestPgSuite(t *testing.T) {
	s := new(PgTestSuite)
	suite.Run(t, s)
}

var _ io.Writer = (*PgTestSuite)(nil)
