// Package daemon manager asyncker daemon
package daemon

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"

	"google.golang.org/grpc"
)

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative daemon.proto

type DaemonManager struct {
	socketPath string
	server     *grpc.Server
}

func NewDaemonManager(socketPath string) *DaemonManager {
	return new(DaemonManager{socketPath: socketPath})
}

func (m *DaemonManager) StartDaemon(registers ...func(*grpc.Server)) (<-chan error, error) {
	old := syscall.Umask(0o177)
	listener, err := net.Listen("unix", m.socketPath)
	syscall.Umask(old)

	if err != nil {
		return nil, fmt.Errorf("create socket listener: %w", err)
	}

	if err := os.Chmod(m.socketPath, 0o600); err != nil {
		listener.Close()
		_ = os.Remove(m.socketPath)
		return nil, fmt.Errorf("securing socket: %w", err)
	}

	m.server = grpc.NewServer()
	for _, register := range registers {
		register(m.server)
	}

	errCh := make(chan error, 1)
	go func() {
		if err := m.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- err
		}
	}()

	return errCh, nil
}

func (m *DaemonManager) StopSignal() {
	if m.server != nil {
		m.server.GracefulStop()
	}
}
