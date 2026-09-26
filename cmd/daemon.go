package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"loadept.com/pkg/asyncker/internal/daemon"
	"loadept.com/pkg/asyncker/internal/task"
)

var up, down, status bool

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage asyncker daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := net.DialTimeout("unix", socketPath, 1*time.Second)
		isRunning := err == nil
		if isRunning {
			conn.Close()
		}

		switch {
		case down || status:
			if !isRunning {
				os.Remove(socketPath)
				return errors.New("no daemon is running")
			}

			client, err := daemon.NewClient(socketPath)
			if err != nil {
				return fmt.Errorf("create daemon client: %w", err)
			}
			defer client.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			switch {
			case down:
				if err := client.StopDaemon(ctx); err != nil {
					return fmt.Errorf("shutdown daemon: %w", err)
				}
				fmt.Println("Daemon shutdown successfully")
			case status:
				status, err := client.DaemonStatus(ctx)
				if err != nil {
					return fmt.Errorf("get status: %w", err)
				}

				w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', 0)
				fmt.Fprintln(w, "Running\tPID\tUptime")
				fmt.Fprintf(
					w,
					"%t\t%d\t%s\n",
					status.GetRunning(),
					status.GetPid(),
					status.GetUptime(),
				)
				w.Flush()
			}
			return nil
		case up:
			if isRunning {
				fmt.Println("A daemon is already running")
				return nil
			}
			if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove orphan socket: %w", err)
			}

			logFile, err := os.OpenFile(daemonLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return fmt.Errorf("open daemon log file: %w", err)
			}
			defer logFile.Close()

			logger := slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			socket := daemon.NewDaemonManager(socketPath)

			stopCh := make(chan struct{}, 1)
			errCh, err := socket.StartDaemon(
				func(s *grpc.Server) {
					daemon.RegisterDaemonServiceServer(s, daemon.NewDaemonServer(logger, stopCh))
				},
				func(s *grpc.Server) {
					task.RegisterTaskServiceServer(s, task.NewTaskServer(logger, logsPath))
				},
			)
			if err != nil {
				return fmt.Errorf("run daemon: %w", err)
			}

			fmt.Println("Daemon running successfully")
			shutdown, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			select {
			case err := <-errCh:
				return err
			case <-shutdown.Done():
				fmt.Println("Shutting down daemon (signal)...")
			case <-stopCh:
				fmt.Println("Shutting down daemon (request)...")
			}
			socket.StopSignal()
			return nil
		}

		return nil
	},
}

func init() {
	daemonCmd.Flags().BoolVar(&up, "up", false, "Start the daemon; this action is idempotent")
	daemonCmd.Flags().BoolVar(&down, "down", false, "Stops the daemon; this action is idempotent")
	daemonCmd.Flags().BoolVar(&status, "status", false, "Stops the daemon; this action is idempotent")
	daemonCmd.MarkFlagsOneRequired("up", "down", "status")
	daemonCmd.MarkFlagsMutuallyExclusive("up", "down", "status")
}
