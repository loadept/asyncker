package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
	"loadept.com/pkg/asyncker/internal/task"
)

var stopCmd = &cobra.Command{
	Use:   "stop task",
	Short: "Stops running task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		conn, err := net.DialTimeout("unix", socketPath, 1*time.Second)
		if err != nil {
			os.Remove(socketPath)
			return errors.New("no daemon is running")
		}
		conn.Close()

		taskName := args[0]

		client, err := task.NewClient(socketPath)
		if err != nil {
			return fmt.Errorf("create task client: %w", err)
		}
		defer client.Close()

		message, err := client.StopTask(ctx, taskName)
		if err != nil {
			return fmt.Errorf("stop task: %w", err)
		}

		fmt.Println("Stopping task:", message)
		return nil
	},
}
