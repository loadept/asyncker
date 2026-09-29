package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"loadept.com/go/asyncker/internal/task"
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

		via, task, err := client.StopTask(ctx, taskName)
		if err != nil {
			return fmt.Errorf("stop task: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', 0)
		defer w.Flush()

		fmt.Fprintln(w, "Task stopped successfully via", via)
		fmt.Fprintln(w, "Name\tPID\tCmdLine\tStatus")
		fmt.Fprintf(
			w,
			"%s\t%d\t%s %s\t%s\n",
			task.GetName(),
			task.GetPid(),
			task.GetCommand(),
			strings.Join(task.GetArgs(), " "),
			task.GetStatus(),
		)
		return nil
	},
}
