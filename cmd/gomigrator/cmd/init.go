package cmd

import (
	"github.com/bojik/sqlmigrator/pkg/config"
	"github.com/spf13/cobra"
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generates config file",
	Long: `Generates config file. For example:

gomigrator init -c ./.gomigrator.yml 
`,
	Run: func(cmd *cobra.Command, args []string) {
		configFile, err := cmd.Flags().GetString(FlagConfig)
		if err != nil {
			cmd.PrintErrln(err.Error())
			return
		}
		if err := config.WriteConfig(configFile); err != nil {
			cmd.PrintErrln(err.Error())
			return
		}
		cmd.Printf("New config file '%s' has been created\n", configFile)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	addConfigFlag(initCmd)
}
