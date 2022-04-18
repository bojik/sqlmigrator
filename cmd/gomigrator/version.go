package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	release   = "UNKNOWN"
	buildDate = "UNKNOWN"
	gitHash   = "UNKNOWN"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := json.NewEncoder(os.Stdout).Encode(struct {
			Release   string
			BuildDate string
			GitHash   string
		}{
			Release:   release,
			BuildDate: buildDate,
			GitHash:   gitHash,
		}); err != nil {
			fmt.Printf("error while decode version info: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	versionCmd.Flags().StringP("dsn", "d", "", "DSN to database")
	versionCmd.Flags().StringP("path", "p", "", "Path to migrations")
}
