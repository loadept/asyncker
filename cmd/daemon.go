package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"loadept.com/pkg/asyncker/internal/daemon"
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
				fmt.Println("No daemon is running")
				if down {
					_ = os.Remove(socketPath)
				}
				return nil
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

			socket := daemon.NewSockerManager(socketPath)

			fmt.Println("Daemon running successfully")
			stopCh, errCh, err := socket.StartDaemon()
			if err != nil {
				return fmt.Errorf("run daemon: %w", err)
			}

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
