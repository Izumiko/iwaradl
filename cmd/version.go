package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Display the current version of iwaradl`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("iwaradl v1.1.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
