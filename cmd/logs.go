package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
	"loadept.com/pkg/asyncker/internal/task"
)

var (
	follow bool
	stderr bool
)

const (
	maxLogSize = 8192
	maxLogBuf  = 4096
)

var logsCmd = &cobra.Command{
	Use:   "logs [flags] task",
	Short: "Show logs task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		taskName := args[0]

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

		outLogs, errLogs, err := client.LogsTask(ctx, taskName)
		if err != nil {
			return fmt.Errorf("listing tasks: %w", err)
		}

		taskLogsPath := outLogs
		if stderr {
			taskLogsPath = errLogs
		}

		file, err := os.Open(taskLogsPath)
		if err != nil {
			return fmt.Errorf("no se pudo abrir pe: %w", err)
		}
		defer file.Close()

		if !follow {
			stat, err := file.Stat()
			if err != nil {
				return err
			}

			var offset int64 = 0
			fileSize := stat.Size()
			if fileSize > maxLogSize {
				offset = fileSize - maxLogSize
			}

			_, err = file.Seek(offset, io.SeekStart)
			if err != nil {
				return err
			}

			if offset > 0 {
				reader := bufio.NewReader(file)
				_, _ = reader.ReadString('\n')
				_, err = io.Copy(os.Stdout, reader)
				return err
			}

			_, err = io.Copy(os.Stdout, file)
			return err
		}

		_, err = file.Seek(0, io.SeekEnd)
		if err != nil {
			return err
		}

		buf := make([]byte, maxLogBuf)
		for {
			if err := ctx.Err(); err != nil {
				return nil
			}

			n, err := file.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])
			}
			if err == io.EOF {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(500 * time.Millisecond):
					continue
				}
			}
			if err != nil {
				return nil
			}
		}
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Format output")
	logsCmd.Flags().BoolVarP(&stderr, "stderr", "e", false, "Show error logs")
}
