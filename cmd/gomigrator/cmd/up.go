package cmd

import (
	"context"

	"github.com/bojik/sqlmigrator/pkg/migrator"
	"github.com/spf13/cobra"
)

// upCmd represents the up command.
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply migrations",
	Long:  `Apply migrations`,
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
		if err := m.ApplyUpSQLMigration(context.Background(), dsn, dir); err != nil {
			//nolint:errorlint
			mErr, ok := err.(migrator.MigrationError)
			if ok {
				cmd.PrintErrln("ERROR", mErr.SQL, mErr.Err.Error(), mErr.File)
				return
			}
			cmd.PrintErrln(err.Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	addConfigFlag(upCmd)
	addPathFlag(upCmd)
	addDsnFlag(upCmd)
	addTypeFlag(upCmd)
}
