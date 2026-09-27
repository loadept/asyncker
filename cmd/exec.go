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
	"loadept.com/pkg/asyncker/internal/task"
)

var name string

var execCmd = &cobra.Command{
	Use:   "exec [flags] command [args...]",
	Short: "Execute a passed command",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		conn, err := net.DialTimeout("unix", socketPath, 1*time.Second)
		if err != nil {
			os.Remove(socketPath)
			return errors.New("no daemon is running")
		}
		conn.Close()

		command := args[0]
		commandArgs := args[1:]

		var taskName *string
		if name != "" {
			taskName = &name
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		envMap := make(map[string]string)
		for _, env := range os.Environ() {
			pair := strings.SplitN(env, "=", 2)
			if len(pair) == 2 {
				envMap[pair[0]] = pair[1]
			}
		}

		client, err := task.NewClient(socketPath)
		if err != nil {
			return fmt.Errorf("create task client: %w", err)
		}
		defer client.Close()

		task, err := client.InvokeTask(ctx, command, commandArgs, taskName, cwd, envMap)
		if err != nil {
			return fmt.Errorf("execute invoke: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', 0)
		defer w.Flush()

		fmt.Fprintln(w, "Command executed successfully")
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

func init() {
	// execCmd.Flags().StringVarP(&output, "output", "o", "", "Redirects stdout and stderr to the specified file")
	execCmd.Flags().StringVar(&name, "name", "", "Assign a name to the task")
	execCmd.Flags().SetInterspersed(false)
}
