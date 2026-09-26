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
	"google.golang.org/protobuf/encoding/protojson"
	"loadept.com/pkg/asyncker/internal/task"
)

var format string

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List registered tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		conn, err := net.DialTimeout("unix", socketPath, 1*time.Second)
		if err != nil {
			os.Remove(socketPath)
			return errors.New("no daemon is running")
		}
		conn.Close()

		client, err := task.NewClient(socketPath)
		if err != nil {
			return fmt.Errorf("create task client: %w", err)
		}
		defer client.Close()

		tasks, err := client.ListTasks(ctx)
		if err != nil {
			return fmt.Errorf("listing tasks: %w", err)
		}

		if format == "" {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', 0)
			fmt.Fprintln(w, "Name\tPID\tCmdLine\tStatus\tExecutedAt")
			for _, t := range tasks {
				fmt.Fprintf(
					w,
					"%s\t%d\t%s %s\t%s\t%s\n",
					t.GetName(),
					t.GetPid(),
					t.GetCommand(),
					strings.Join(t.GetArgs(), " "),
					t.GetStatus(),
					time.Unix(t.GetExecutedAt(), 0).Format("2006-01-02 15:04:05"),
				)
			}
			w.Flush()
			return nil
		}

		switch format {
		case "json":
			for _, t := range tasks {
				out, err := protojson.Marshal(t)
				if err != nil {
					return fmt.Errorf("formating tasks: %w", err)
				}
				fmt.Printf("%s\n", out)
			}
		default:
			return fmt.Errorf("unknown format: %s", format)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&format, "format", "", "List executed tasks")
}
