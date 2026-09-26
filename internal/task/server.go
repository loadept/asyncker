package task

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"loadept.com/pkg/asyncker/internal/namegenerator"
)

type TaskServer struct {
	UnimplementedTaskServiceServer
	logger   *slog.Logger
	logsPath string
	mu       sync.RWMutex
	tasks    map[string]*Task
}

func NewTaskServer(logger *slog.Logger, logsPath string) *TaskServer {
	return new(TaskServer{
		logger:   logger,
		logsPath: logsPath,
		tasks:    make(map[string]*Task),
	})
}

func (s *TaskServer) InvokeTask(ctx context.Context, req *InvokeTaskRequest) (*InvokeTaskResponse, error) {
	command := req.Command
	args := req.Args
	taskName, err := s.reserveTaskName(req.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "reserve name: %v", err)
	}

	outFilePath := filepath.Join(s.logsPath, fmt.Sprintf("%s_out.log", taskName))
	outFile, err := os.OpenFile(outFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		s.releaseTaskName(taskName)
		return nil, status.Errorf(codes.Internal, "open out log file: %v", err)
	}

	errFilePath := filepath.Join(s.logsPath, fmt.Sprintf("%s_err.log", taskName))
	errFile, err := os.OpenFile(errFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		outFile.Close()
		os.Remove(outFilePath)
		s.releaseTaskName(taskName)
		return nil, status.Errorf(codes.Internal, "open err log file: %v", err)
	}

	cmd := exec.Command(req.Command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = outFile
	cmd.Stderr = errFile

	if err := cmd.Start(); err != nil {
		outFile.Close()
		errFile.Close()
		os.Remove(outFilePath)
		os.Remove(errFilePath)
		s.releaseTaskName(taskName)
		return nil, fmt.Errorf("execute command: %w", err)
	}

	cmdStr := strings.TrimSpace(fmt.Sprintf("%s %s", command, strings.Join(args, " ")))
	pid := cmd.Process.Pid

	task := &Task{
		Name:       taskName,
		Command:    command,
		Args:       args,
		Pid:        int64(pid),
		Status:     TaskStatus_TASK_STATUS_RUNNING,
		ExecutedAt: time.Now().Unix(),
	}

	s.mu.Lock()
	s.tasks[taskName] = task
	s.mu.Unlock()

	s.logger.Info("command executed successfully", "name", taskName, "command", cmdStr, "pid", pid)
	go func(c *exec.Cmd, out, errF *os.File) {
		defer out.Close()
		defer errF.Close()

		waitErr := c.Wait()

		s.mu.Lock()
		defer s.mu.Unlock()

		t, exists := s.tasks[taskName]
		if !exists || t == nil {
			return
		}

		if waitErr != nil {
			t.Status = TaskStatus_TASK_STATUS_FAILED
			s.logger.Info("command finished with error", "name", taskName, "command", cmdStr, "pid", pid, "err", waitErr)
		} else {
			t.Status = TaskStatus_TASK_STATUS_SUCCEEDED
			s.logger.Info("command finished successfully", "name", taskName, "command", cmdStr, "pid", pid)
		}
	}(cmd, outFile, errFile)

	return &InvokeTaskResponse{Task: task}, nil
}

func (s *TaskServer) ListTasks(ctx context.Context, _ *emptypb.Empty) (*ListTasksResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	taskList := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		taskList = append(taskList, task)
	}

	return &ListTasksResponse{Tasks: taskList}, nil
}

func (s *TaskServer) reserveTaskName(reqName *string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	useSufix := false
	if reqName != nil {
		if _, exists := s.tasks[*reqName]; exists {
			return "", fmt.Errorf("task name %q already in use", *reqName)
		}
		s.tasks[*reqName] = nil
		return *reqName, nil
	}

	for {
		newName := namegenerator.GenerateName(useSufix)
		if _, exists := s.tasks[newName]; exists {
			useSufix = true
			continue
		}
		s.tasks[newName] = nil
		return newName, nil
	}
}

func (s *TaskServer) releaseTaskName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, name)
}
