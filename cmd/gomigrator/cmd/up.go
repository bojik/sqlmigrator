package cmd

import (
	"context"
	"fmt"

	"github.com/bojik/sqlmigrator/internal/config"
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
		m := migrator.New(getLogger(cmd))
		results, err := m.ApplyUpSQLMigration(context.Background(), config.GetDsn(), config.GetPath())
		if err != nil {
			cmd.PrintErrln(err.Error())
		}
		for _, result := range results {
			cmd.Println(fmt.Sprintf("%d|%s", result.Version, result.Status.String()))
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
