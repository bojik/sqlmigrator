package cmd

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

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Returns version information about application",
	Long:  `Returns version information about application`,
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
}
