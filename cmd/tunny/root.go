package main

import (
	"github.com/spf13/cobra"
)

var (
	daemonMode bool
	daemonPID  string
	daemonLog  string

	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "tunny",
	Short: "Location-aware network tunneling",
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.yaml", "Path to config file")

	// daemon mode flags
	rootCmd.PersistentFlags().BoolVarP(&daemonMode, "daemon", "d", false, "Run in background")
	rootCmd.PersistentFlags().StringVarP(&daemonPID, "pid", "p", "/var/run/tunny.pid", "Path to PID file")
	rootCmd.PersistentFlags().StringVarP(&daemonLog, "log", "l", "/var/log/tunny.log", "Path to log file")

	rootCmd.AddCommand(providerCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(tunnelCmd)
	rootCmd.AddCommand(daemonCmd)
}
