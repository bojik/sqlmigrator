package cmd

import (
	"context"

	"github.com/bojik/sqlmigrator/pkg/migrator"
	"github.com/spf13/cobra"
)

// downCmd represents the down command.
var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Down migrations",
	Long:  `Down migrations`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := loadConfigData(cmd); err != nil {
			cmd.PrintErrln(err.Error())
			return
		}
		m := migrator.New(cmd.OutOrStdout())
		dsn, err := cmd.Flags().GetString(FlagDsn)
		if err != nil {
			cmd.PrintErrln(err.Error())
			return
		}
		dir, err := cmd.Flags().GetString(FlagPath)
		if err != nil {
			cmd.PrintErrln(err.Error())
			return
		}
		if err := m.ApplyDownSQLMigration(context.Background(), dsn, dir); err != nil {
			cmd.PrintErrln(err.Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
	addConfigFlag(downCmd)
	addPathFlag(downCmd)
	addDsnFlag(downCmd)
	addTypeFlag(downCmd)
}
