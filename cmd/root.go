// Package cmd provides command to interact with asyncker
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	configDirPath string
	logsDirPath   string
	socketPath    string
	tasksFilePath string
	selfPath      string
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

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(stopCmd)
}

func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("open home directory: %v\n", err)
		return
	}

	configDirName := ".asyncker"
	logsDirName := "logs"
	tasksFileName := "tasks.gob"
	sockerName := "daemon.sock"

	configDirPath = filepath.Join(home, configDirName)
	logsDirPath = filepath.Join(configDirPath, logsDirName)
	socketPath = filepath.Join(configDirPath, sockerName)
	tasksFilePath = filepath.Join(configDirPath, tasksFileName)

	if err := os.MkdirAll(logsDirPath, 0o755); err != nil {
		fmt.Printf("create project directory: %v\n", err)
		return
	}

	if _, err := os.Stat(tasksFilePath); os.IsNotExist(err) {
		file, err := os.OpenFile(tasksFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			fmt.Printf("create project file: %v\n", err)
			return
		}
		defer file.Close()
	}

	selfPath, err = os.Executable()
	if err != nil {
		fmt.Printf("get self path: %v\n", err)
		return
	}
}
