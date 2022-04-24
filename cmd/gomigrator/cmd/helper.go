package cmd

import (
	"errors"
	"os"

	"github.com/bojik/sqlmigrator/internal/config"
	"github.com/spf13/cobra"
)

const (
	FlagConfig = "config"
	FlagFormat = config.KeyFormat
	FlagPath   = config.KeyPath
	FlagDsn    = config.KeyDsn
)

func loadConfigData(cmd *cobra.Command) error {
	configFile, err := cmd.Flags().GetString(FlagConfig)
	if err != nil {
		return err
	}
	if _, err := os.Stat(configFile); err != nil {
		configFile = ""
	}
	if err := config.Load(configFile, cmd.Flags()); err != nil {
		return err
	}
	if typeFlag := cmd.Flags().Lookup(FlagFormat); typeFlag != nil {
		v := typeFlag.Value.String()
		if v != config.FormatGo.String() && v != config.FormatSQL.String() {
			return errors.New(
				"invalid value of flag --" + FlagFormat + ". expected: " + config.FormatSQL.String() +
					"|" + config.FormatGo.String(),
			)
		}
	}
	return nil
}

func addConfigFlag(command *cobra.Command) {
	command.Flags().StringP(FlagConfig, "c", config.DefaultConfigFile, "Path to config file")
}

func addPathFlag(command *cobra.Command) {
	command.Flags().StringP(FlagPath, "p", "", "Path to migration")
}

func addTypeFlag(command *cobra.Command) {
	command.Flags().StringP(
		FlagFormat,
		"t",
		config.FormatSQL.String(),
		"Type of migration. Expected: "+config.FormatSQL.String()+"|"+config.FormatGo.String(),
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
