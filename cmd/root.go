// Package cmd provides command to interact with asyncker
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	configPath    string
	logsPath      string
	daemonLogFile string
	socketPath    string
	tasksPath     string
	// selfPath      string
)

var rootCmd = &cobra.Command{
	Use:   "asyncker",
	Short: "Tool for executing commands asynchronously.",
	Long: `asyncker is a CLI written in Go for
executing and managing commands asynchronously.`,
	SilenceUsage: true,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
}

func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(logsCmd)
}

func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("open home directory: %v\n", err)
		return
	}

	configPath = filepath.Join(home, ".asyncker")
	logsPath = filepath.Join(configPath, "logs")
	socketPath = filepath.Join(configPath, "daemon.sock")
	tasksPath = filepath.Join(configPath, "tasks.gob")
	daemonLogFile = filepath.Join(configPath, "daemon.log")

	if err := os.MkdirAll(logsPath, 0o750); err != nil {
		fmt.Printf("create project directory: %v\n", err)
		return
	}

	if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
		file, err := os.OpenFile(tasksPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			fmt.Printf("create project file: %v\n", err)
			return
		}
		defer file.Close()
	}

	// selfPath, err = os.Executable()
	// if err != nil {
	// 	fmt.Printf("get self path: %v\n", err)
	// 	return
	// }
}
