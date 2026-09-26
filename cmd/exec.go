package cmd

import (
	"fmt"
	"net"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"loadept.com/pkg/asyncker/internal/namegenerator"
	"loadept.com/pkg/asyncker/internal/task"
)

var name string

var execCmd = &cobra.Command{
	Use:   "exec [flags] command [args...]",
	Short: "Execute a passed command",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := net.DialTimeout("unix", socketPath, 1*time.Second)
		if err != nil {
			fmt.Println("No daemon is running")
			if down {
				os.Remove(socketPath)
			}
			return nil
		}
		conn.Close()

		command := args[0]
		commandArgs := args[1:]
		var taskName *string
		if name != "" {
			taskName = &name
		}

		client, err := task.NewClient(socketPath)
		if err != nil {
			return fmt.Errorf("create task client: %w", err)
		}
		defer client.Close()

		task, err := client.InvokeTask(cmd.Context(), &task.InvokeTaskRequest{
			Command: command,
			Args:    commandArgs,
			Name:    taskName,
		})
		if err != nil {
			return fmt.Errorf("execute invoke: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', 0)
		defer w.Flush()

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

func genTaskName(tasks []Task) string {
	useSufix := false
outer:
	for {
		newName := namegenerator.GenerateName(useSufix)
		for _, t := range tasks {
			if t.Name == newName {
				useSufix = true
				continue outer
			}
		}
		return newName
	}
}

func genTaskID(tasks []Task) int {
	if len(tasks) > 0 {
		return tasks[len(tasks)-1].ID + 1
	}
	return 0
}
