package cmd

import (
	"github.com/spf13/cobra"
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Applyes migrations",
	Long:  `Applyes migrations`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := loadConfigData(cmd); err != nil {
			cmd.PrintErrln(err.Error())
			return
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
