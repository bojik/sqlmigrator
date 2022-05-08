package cmd

import (
	"strings"

	config2 "github.com/bojik/sqlmigrator/pkg/config"
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
		mig := migrator.New(getLogger(cmd))
		if config2.GetType() == config2.FormatSQL.String() {
			up, down, err := mig.CreateSQLMigration(config2.GetPath(), strings.Join(args, "_"))
			if err != nil {
				cmd.PrintErrln(err.Error())
				return
			}
			cmd.Printf("Created up sql migration: %s\n", up)
			cmd.Printf("Created down sql migration: %s\n", down)
			return
		}
		if config2.GetType() == config2.FormatGo.String() {
			file, err := mig.CreateGoMigration(config2.GetPath(), strings.Join(args, "_"))
			if err != nil {
				cmd.PrintErrln(err.Error())
				return
			}
			cmd.Printf("Created GO migration: %s\n", file)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	addConfigFlag(createCmd)
	addPathFlag(createCmd)
	addTypeFlag(createCmd)
}
