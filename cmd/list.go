package cmd

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var format string

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List registered tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		tasks, err := LoadTasks(tasksFilePath)
		if err != nil {
			return fmt.Errorf("listing tasks: %w", err)
		}

		if format == "" {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', 0)
			fmt.Fprintln(w, "ID\tName\tPID\tCmdLine\tExecutedAt")
			for _, t := range tasks {
				fmt.Fprintf(
					w,
					"%d\t%s\t%d\t%s %s\t%s\n",
					t.ID,
					t.Name,
					t.PID,
					t.CmdLine.Command,
					strings.Join(t.CmdLine.Args, " "),
					time.Unix(t.ExecutedAt, 0).Format("2006-01-02 15:04:05"),
				)
			}
			w.Flush()
			return nil
		}

		switch format {
		case "json":
			for _, t := range tasks {
				out, err := json.Marshal(t)
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
