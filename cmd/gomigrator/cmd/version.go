package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed _release.txt
var release string

//go:embed _build_date.txt
var buildDate string

//go:embed _githash.txt
var gitHash string

//go:generate sh version.sh
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
			Release:   strings.TrimRight(release, "\n"),
			BuildDate: strings.TrimRight(buildDate, "\n"),
			GitHash:   strings.TrimRight(gitHash, "\n"),
		}); err != nil {
			fmt.Printf("error while decode version info: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
