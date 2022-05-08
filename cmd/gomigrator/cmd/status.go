package cmd

import (
	"fmt"
	"time"

	"github.com/bojik/sqlmigrator/pkg/config"
	"github.com/bojik/sqlmigrator/pkg/migrator"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command.
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Shows migration statuses.",
	Long:  `Shows migration statuses.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := loadConfigData(cmd); err != nil {
			cmd.PrintErrln(err.Error())
			return
		}
		m := migrator.New(getLogger(cmd))
		rows, err2 := m.SelectStatuses(cmd.Context(), config.GetDsn())
		if err2 != nil {
			cmd.PrintErrln(err2.Error())
			return
		}
		for _, row := range rows {
			cmd.PrintErrln(
				fmt.Sprintf(
					"%d|%s|%s",
					row.Version,
					row.ExecutedAt.Format(time.RFC3339),
					row.Status.String(),
				),
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	addConfigFlag(statusCmd)
	addDsnFlag(statusCmd)
	addTableFlag(statusCmd)
}
