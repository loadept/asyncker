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
			fmt.Fprintln(w, "Name\tPID\tCmdLine\tStatus\tUptime\tMemUsage\tNumThreads")
			for _, task := range tasks {
				fmt.Fprintf(
					w,
					"%s\t%d\t%s %s\t%s\t%s\t%s\t%d\n",
					task.GetName(),
					task.GetPid(),
					task.GetCommand(),
					strings.Join(task.GetArgs(), " "),
					task.GetStatus(),
					time.Since(time.Unix(task.GetExecutedAt(), 0)).String(),
					fmt.Sprintf("%.2fmb", float64(task.GetMemUsage())/(1024*1024)),
					task.GetNumThreads(),
				)
			}
			w.Flush()
			return nil
		}

		switch format {
		case "json":
			m := protojson.MarshalOptions{
				EmitUnpopulated: true,
				UseEnumNumbers:  false,
			}
			for _, task := range tasks {
				out, err := m.Marshal(task)
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
