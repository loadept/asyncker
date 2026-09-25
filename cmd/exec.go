package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"loadept.com/pkg/asyncker/internal/namegenerator"
)

var (
	output   string
	taskName string
)

var execCmd = &cobra.Command{
	Use:   "exec [flags] command [args...]",
	Short: "Execute a passed command",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		cmdLine := CmdLine{
			Command: args[0],
			Args:    args[1:],
		}

		cmd := exec.Command(cmdLine.Command, cmdLine.Args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

		var (
			stdOut *os.File
			stdErr *os.File
		)
		if output != "" {
			absPath, err := filepath.Abs(output)
			if err != nil {
				panic(err)
			}
			outFile, err := os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return err
			}
			stdOut = outFile
			stdErr = outFile
		} else {
			outFileName := fmt.Sprintf("%s_out.log", cmdLine.Command)
			outFilePath := filepath.Join(logsPath, outFileName)
			outFile, err := os.OpenFile(outFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return err
			}

			errFileName := fmt.Sprintf("%s_err.log", cmdLine.Command)
			errFilePath := filepath.Join(logsPath, errFileName)
			errFile, err := os.OpenFile(errFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return err
			}
			stdOut = outFile
			stdErr = errFile
		}
		cmd.Stdout = stdOut
		cmd.Stderr = stdErr

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("execute command: %w", err)
		}

		tasks, err := LoadTasks(tasksPath)
		if err != nil {
			return fmt.Errorf("load tasks: %w", err)
		}

		if taskName == "" {
			taskName = genTaskName(tasks)
		}
		newTask := Task{
			ID:         genTaskID(tasks),
			Name:       taskName,
			PID:        cmd.Process.Pid,
			CmdLine:    cmdLine,
			ExecutedAt: time.Now().Unix(),
			Logs: Logs{
				Output: stdOut.Name(),
				Error:  stdErr.Name(),
			},
		}
		tasks = append(tasks, newTask)

		if err := SaveTasks(tasksPath, tasks); err != nil {
			return fmt.Errorf("save task: %w", err)
		}

		fmt.Printf("Command executed with PID: %d\n", newTask.PID)
		return nil
	},
}

func init() {
	execCmd.Flags().StringVarP(&output, "output", "o", "", "Redirects stdout and stderr to the specified file")
	execCmd.Flags().StringVar(&taskName, "name", "", "Assign a name to the task")
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
