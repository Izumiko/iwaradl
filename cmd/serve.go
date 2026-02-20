package cmd

import (
	"iwaradl/server"

	"github.com/spf13/cobra"
)

var port int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start iwara downloading daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.RunServer(port)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVarP(&port, "port", "p", 23456, "listen port for daemon mode")
}
