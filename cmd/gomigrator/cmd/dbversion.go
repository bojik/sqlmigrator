package cmd

import (
	"github.com/bojik/sqlmigrator/pkg/config"
	"github.com/bojik/sqlmigrator/pkg/migrator"
	"github.com/spf13/cobra"
)

// dbversionCmd represents the dbversion command.
var dbversionCmd = &cobra.Command{
	Use:   "dbversion",
	Short: "Shows last migration version",
	Long:  `Shows last migration version`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := loadConfigData(cmd); err != nil {
			cmd.PrintErrln(err.Error())
			return
		}
		m := migrator.New(getLogger(cmd))
		version, err := m.SelectDBVersion(cmd.Context(), config.GetDsn())
		if err != nil {
			cmd.PrintErrln(err.Error())
			return
		}
		cmd.Println(version)
	},
}

func init() {
	rootCmd.AddCommand(dbversionCmd)
	addConfigFlag(dbversionCmd)
	addDsnFlag(dbversionCmd)
}
