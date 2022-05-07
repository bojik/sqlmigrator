/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/bojik/sqlmigrator/pkg/config"
	"github.com/bojik/sqlmigrator/pkg/migrator"
	"github.com/spf13/cobra"
)

// redoCmd represents the redo command.
var redoCmd = &cobra.Command{
	Use:   "redo",
	Short: "Redo last migration",
	Long:  `Redo last migration`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := loadConfigData(cmd); err != nil {
			cmd.PrintErrln(err.Error())
			return
		}
		m := migrator.New(getLogger(cmd))
		results, err := m.ApplyRedoSQLMigration(cmd.Context(), config.GetDsn(), config.GetPath())
		if err != nil {
			cmd.PrintErrln(err.Error())
		}
		cmd.Println(formatResults(results))
	},
}

func init() {
	rootCmd.AddCommand(redoCmd)
	addConfigFlag(redoCmd)
	addPathFlag(redoCmd)
	addDsnFlag(redoCmd)
	addTypeFlag(redoCmd)
}
