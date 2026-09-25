package task

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"google.golang.org/protobuf/types/known/emptypb"
	"loadept.com/pkg/asyncker/internal/namegenerator"
)

type TaskServer struct {
	UnimplementedTaskServiceServer
	logger   *slog.Logger
	logsPath string
}

func NewTaskServer(logger *slog.Logger, logsPath string) *TaskServer {
	return new(TaskServer{
		logger:   logger,
		logsPath: logsPath,
	})
}

func (s *TaskServer) InvokeTask(ctx context.Context, req *InvokeTaskRequest) (*InvokeTaskResponse, error) {
	command := req.Command
	args := req.Args

	cmd := exec.Command(req.Command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	taskName := namegenerator.GenerateName(false)
	if req.Name != nil {
		taskName = *req.Name
	}

	outFileName := fmt.Sprintf("%s_out.log", taskName)
	outFilePath := filepath.Join(s.logsPath, outFileName)
	outFile, err := os.OpenFile(outFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	errFileName := fmt.Sprintf("%s_err.log", taskName)
	errFilePath := filepath.Join(s.logsPath, errFileName)
	errFile, err := os.OpenFile(errFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		outFile.Close()
		return nil, err
	}

	cmd.Stdout = outFile
	cmd.Stderr = errFile

	if err := cmd.Start(); err != nil {
		outFile.Close()
		errFile.Close()
		return nil, fmt.Errorf("execute command: %w", err)
	}

	cmdStr := strings.TrimSpace(fmt.Sprintf("%s %s", command, strings.Join(args, " ")))
	pid := cmd.Process.Pid

	s.logger.Info("command executed successfully", "name", taskName, "command", cmdStr, "pid", pid)
	go func(c *exec.Cmd, out *os.File, errF *os.File) {
		defer out.Close()
		defer errF.Close()

		if err := c.Wait(); err != nil {
			s.logger.Info("command finished with error", "name", taskName, "command", cmdStr, "pid", pid, "err", err)
			return
		}
		s.logger.Info("command finished successfully", "name", taskName, "command", cmdStr, "pid", pid)
	}(cmd, outFile, errFile)

	return &InvokeTaskResponse{
		Task: &Task{
			Id:      0,
			Name:    taskName,
			Command: command,
			Args:    args,
			Pid:     int64(pid),
			Status:  true,
		},
	}, nil
}

func (s *TaskServer) ListTasks(ctx context.Context, _ *emptypb.Empty) (*ListTasksResponse, error) {
	return nil, nil
}
