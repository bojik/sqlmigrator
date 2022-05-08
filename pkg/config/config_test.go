package config

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestConfigFile(t *testing.T) {
	cfg := New()
	err := cfg.LoadFromFile("./testdata/config.yml")
	require.Nil(t, err)
	require.Equal(t, "dsn", cfg.GetDsn())
	require.Equal(t, "path", cfg.GetPath())
	require.Equal(t, "sql", cfg.GetType())
	require.Equal(t, "versions", cfg.GetTable())
}

func TestCommandLine(t *testing.T) {
	cfg := New()
	err := cfg.LoadFromFile("./testdata/config.yml")
	require.Nil(t, err)

	flagSet := pflag.NewFlagSet("test", pflag.PanicOnError)
	flagSet.StringP("dsn", "d", "", "")
	flagSet.StringP("path", "p", "", "")
	flagSet.StringP("type", "t", "", "")
	flagSet.StringP("table", "m", "", "")
	err = flagSet.Parse([]string{"--dsn", "dsn2", "--path", "path2", "--type", "go", "--table", "ver2"})
	require.Nil(t, err)
	err = cfg.LoadFromCommandLine(flagSet)
	require.Nil(t, err)

	require.Equal(t, "dsn2", cfg.GetDsn())
	require.Equal(t, "path2", cfg.GetPath())
	require.Equal(t, "go", cfg.GetType())
	require.Equal(t, "ver2", cfg.GetTable())
}

func TestEnv(t *testing.T) {
	cfg := New()
	err := cfg.LoadFromFile("./testdata/config.yml")
	require.Nil(t, err)

	err = os.Setenv("GOMIGRATOR_DSN", "dsn3")
	require.Nil(t, err)
	err = os.Setenv("GOMIGRATOR_PATH", "path3")
	require.Nil(t, err)
	err = os.Setenv("GOMIGRATOR_TYPE", "go3")
	require.Nil(t, err)
	err = os.Setenv("GOMIGRATOR_TABLE", "ver3")
	require.Nil(t, err)
	err = cfg.LoadFromEnv()
	require.Nil(t, err)
	require.Equal(t, "dsn3", cfg.GetDsn())
	require.Equal(t, "path3", cfg.GetPath())
	require.Equal(t, "go3", cfg.GetType())
	require.Equal(t, "ver3", cfg.GetTable())
}

func TestDump(t *testing.T) {
	cfg := New()
	testCfg := "./testdata/testcfg.yml"
	err := cfg.WriteConfig(testCfg)
	require.Nil(t, err)
	defer func() {
		if _, err := os.Stat(testCfg); err == nil {
			_ = os.Remove(testCfg)
		}
	}()
	actual, err := ioutil.ReadFile(testCfg)
	require.Nil(t, err)
	expected := []byte(`dsn: ""
path: ""
table: dbmigrator_version
type: ""
`)
	require.Equal(t, expected, actual)
	err = cfg.WriteConfig(testCfg)
	require.Error(t, err)
}
