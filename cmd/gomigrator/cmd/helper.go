package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	config2 "github.com/bojik/sqlmigrator/pkg/config"
	"github.com/bojik/sqlmigrator/pkg/db"
	"github.com/bojik/sqlmigrator/pkg/migrator"
	"github.com/spf13/cobra"
)

const (
	FlagConfig  = "config"
	FlagVerbose = "verbose"
	FlagFormat  = config2.KeyFormat
	FlagPath    = config2.KeyPath
	FlagDsn     = config2.KeyDsn
	FlagTable   = config2.KeyTable
)

func loadConfigData(cmd *cobra.Command) error {
	configFile, err := cmd.Flags().GetString(FlagConfig)
	if err != nil {
		return err
	}
	if _, err := os.Stat(configFile); err != nil {
		configFile = ""
	}
	if err := config2.Load(configFile, cmd.Flags()); err != nil {
		return err
	}
	if typeFlag := cmd.Flags().Lookup(FlagFormat); typeFlag != nil {
		v := typeFlag.Value.String()
		if v != config2.FormatGo.String() && v != config2.FormatSQL.String() {
			return errors.New(
				"invalid value of flag --" + FlagFormat + ". expected: " + config2.FormatSQL.String() +
					"|" + config2.FormatGo.String(),
			)
		}
	}
	return nil
}

func getLogger(cmd *cobra.Command) io.Writer {
	empty := emptyWriter{}
	b, err := cmd.Flags().GetBool(FlagVerbose)
	if err != nil {
		return empty
	}
	if b {
		return cmd.OutOrStdout()
	}
	return empty
}

func addConfigFlag(command *cobra.Command) {
	command.Flags().StringP(FlagConfig, "c", config2.DefaultConfigFile, "Path to config file")
}

func addPathFlag(command *cobra.Command) {
	command.Flags().StringP(FlagPath, "p", "", "Path to migration")
}

func addTypeFlag(command *cobra.Command) {
	command.Flags().StringP(
		FlagFormat,
		"t",
		config2.FormatSQL.String(),
		"Type of migration. Expected: "+config2.FormatSQL.String()+"|"+config2.FormatGo.String(),
	)
}

func addDsnFlag(command *cobra.Command) {
	command.Flags().StringP(
		FlagDsn,
		"d",
		"",
		"DSN to database",
	)
}

func addTableFlag(command *cobra.Command) {
	command.Flags().StringP(
		FlagTable,
		"m",
		"",
		"Table name for meta information",
	)
}

type emptyWriter struct{}

func (emptyWriter) Write([]byte) (n int, err error) {
	return 0, nil
}

var _ io.Writer = (*emptyWriter)(nil)

func formatResults(rs []*migrator.Result) string {
	sb := &strings.Builder{}
	for _, r := range rs {
		sb.WriteString(formatResult(r))
	}
	return sb.String()
}

func formatResult(r *migrator.Result) string {
	s := fmt.Sprintf("Migration %s to version %d:\n%s\n", r.Type.String(), r.Version, r.SQL)
	if r.Status == db.Error {
		s = fmt.Sprintf("%sError: %s\n", s, r.Err.Error())
	}
	if r.Status == db.Success {
		s = fmt.Sprintf("%sOK\n", s)
	}
	return s
}
