package cmd

import (
	"fmt"

	"github.com/bojik/sqlmigrator/internal/config"
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
		m := migrator.New(getLogger(cmd))
		results, err1 := m.ApplyDownSQLMigration(cmd.Context(), config.GetDsn(), config.GetPath())
		if err1 != nil {
			cmd.PrintErrln(err1.Error())
		}
		for _, result := range results {
			cmd.Println(fmt.Sprintf("%d|%s", result.Version, result.Status.String()))
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
