package cmd

import (
	"strings"

	"github.com/bojik/sqlmigrator/internal/config"
	"github.com/bojik/sqlmigrator/pkg/migrator"
	"github.com/spf13/cobra"
)

// createCmd represents the create command.
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates new migration file.",
	Long:  `Creates new migration file.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := loadConfigData(cmd); err != nil {
			cmd.PrintErrln(err.Error())
			return
		}
		mig := migrator.New(cmd.OutOrStdout())
		format, err := cmd.Flags().GetString(FlagFormat)
		if err != nil {
			cmd.PrintErrln(err.Error())
			return
		}
		if format == config.FormatSQL.String() {
			up, down, err := mig.CreateSQLMigration(config.GetPath(), strings.Join(args, "_"))
			if err != nil {
				cmd.PrintErrln(err.Error())
				return
			}
			cmd.Printf("Creating up sql migration: %s\n", up)
			cmd.Printf("Creating down sql migration: %s\n", down)
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	addConfigFlag(createCmd)
	addPathFlag(createCmd)
	addTypeFlag(createCmd)
}
