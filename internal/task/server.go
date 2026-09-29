package task

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"loadept.com/go/asyncker/internal/namegenerator"
)

const killTermTimeout = 10 * time.Second

var pageSize = uint64(os.Getpagesize())

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
	outFile, err := os.OpenFile(outFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		s.releaseTaskName(taskName)
		return nil, status.Errorf(codes.Internal, "open out log file: %v", err)
	}

	errFilePath := filepath.Join(s.logsPath, fmt.Sprintf("%s_err.log", taskName))
	errFile, err := os.OpenFile(errFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		outFile.Close()
		os.Remove(outFilePath)
		s.releaseTaskName(taskName)
		return nil, status.Errorf(codes.Internal, "open err log file: %v", err)
	}

	// #nosec G204 -- command received only via local Unix socket, not exposed to the network
	// nolint:noctx // intentional fire-and-forget
	cmd := exec.Command(req.Command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if req.WorkingDir != "" {
		cmd.Dir = req.WorkingDir
	}
	if len(req.EnvVars) > 0 {
		envSlice := make([]string, 0, len(req.EnvVars))
		for k, v := range req.EnvVars {
			envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = envSlice
	}
	cmd.Stdout = outFile
	cmd.Stderr = errFile

	if err := cmd.Start(); err != nil {
		outFile.Close()
		errFile.Close()
		os.Remove(outFilePath)
		os.Remove(errFilePath)
		s.releaseTaskName(taskName)
		return nil, status.Errorf(codes.Internal, "execute command: %v", err)
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

		if waitErr == nil {
			t.Status = TaskStatus_TASK_STATUS_SUCCEEDED
			s.logger.Info("command finished successfully", "name", taskName, "command", cmdStr, "pid", pid)
			return
		}

		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				sig := status.Signal()
				if sig == syscall.SIGTERM || sig == syscall.SIGINT {
					t.Status = TaskStatus_TASK_STATUS_CANCELLED
					s.logger.Info("command cancelled by signal", "name", taskName, "signal", sig.String())
					return
				}
			}
		}

		t.Status = TaskStatus_TASK_STATUS_FAILED
		s.logger.Info("command finished with error", "name", taskName, "command", cmdStr, "pid", pid, "err", waitErr)
	}(cmd, outFile, errFile)

	return &InvokeTaskResponse{Task: task}, nil
}

func (s *TaskServer) ListTasks(ctx context.Context, _ *emptypb.Empty) (*ListTasksResponse, error) {
	taskList := make([]*Task, 0, len(s.tasks))
	s.mu.RLock()
	for _, task := range s.tasks {
		taskList = append(taskList, task)
	}
	s.mu.RUnlock()

	slices.SortFunc(taskList, func(a, b *Task) int {
		return cmp.Compare(a.GetExecutedAt(), b.GetExecutedAt())
	})

	for _, task := range taskList {
		if task.GetStatus() != TaskStatus_TASK_STATUS_RUNNING {
			task.MemUsage = 0
			task.NumThreads = 0
			continue
		}

		statmData, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", task.GetPid()))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "read proc statm")
		}
		statmFields := bytes.Fields(statmData)
		if len(statmFields) < 2 {
			return nil, status.Errorf(codes.Internal, "statm does not contain enough fields")
		}
		rssPages, _ := strconv.ParseUint(string(statmFields[1]), 10, 64)
		task.MemUsage = rssPages * pageSize

		statData, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", task.GetPid()))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "read proc stat: %v", err)
		}
		lastParen := bytes.LastIndexByte(statData, ')')
		if lastParen == -1 || len(statData) <= lastParen+2 {
			return nil, status.Errorf(codes.Internal, "invalid format of stat")
		}
		statFields := bytes.Fields(statData[lastParen+2:])
		if len(statFields) < 18 {
			return nil, status.Errorf(codes.Internal, "stat does not contain enough fields")
		}
		threads, _ := strconv.ParseUint(string(statFields[17]), 10, 32)
		task.NumThreads = uint32(threads)
	}

	return &ListTasksResponse{Tasks: taskList}, nil
}

func (s *TaskServer) StopTask(ctx context.Context, req *StopTaskRequest) (*StopTaskResponse, error) {
	taskName := req.GetName()

	s.mu.RLock()
	task, ok := s.tasks[taskName]
	s.mu.RUnlock()

	if !ok || task == nil {
		return nil, status.Errorf(codes.NotFound, "task %s not found", taskName)
	}
	if task.GetStatus() != TaskStatus_TASK_STATUS_RUNNING {
		return nil, status.Errorf(codes.FailedPrecondition, "task %q is not running (status: %s)", req.Name, task.Status)
	}

	pid := int(task.GetPid())

	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil && err != syscall.ESRCH {
		return nil, status.Errorf(codes.Internal, "terminate task: %v", err)
	}
	if !pollUntilDead(pid, killTermTimeout) {
		if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
			s.logger.Warn("failed to send SIGKILL", "pid", pid, "err", err)
		}
		pollUntilDead(pid, 2*time.Second)

		return &StopTaskResponse{
			Stopped: true,
			Via:     syscall.SIGKILL.String(),
			Task:    task,
		}, nil
	}

	return &StopTaskResponse{
		Stopped: true,
		Via:     syscall.SIGTERM.String(),
		Task:    task,
	}, nil
}

func (s *TaskServer) LogsTask(ctx context.Context, req *LogsTaskRequest) (*LogsTaskResponse, error) {
	taskName := req.GetName()

	s.mu.RLock()
	if _, ok := s.tasks[taskName]; !ok {
		return nil, status.Errorf(codes.NotFound, "task %s not found", taskName)
	}
	s.mu.RUnlock()

	outLogs := filepath.Join(s.logsPath, fmt.Sprintf("%s_out.log", taskName))
	errLogs := filepath.Join(s.logsPath, fmt.Sprintf("%s_err.log", taskName))

	return &LogsTaskResponse{
		StdoutPath: outLogs,
		StderrPath: errLogs,
	}, nil
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

func pollUntilDead(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
