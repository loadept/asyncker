// Package daemon manager asyncker daemon
package daemon

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

type SockerManager struct {
	socketPath string
	server     *grpc.Server
}

func NewSockerManager(socketPath string) *SockerManager {
	return new(SockerManager{socketPath: socketPath})
}

func (m *SockerManager) StartDaemon() (<-chan struct{}, <-chan error, error) {
	old := syscall.Umask(0o177)
	listener, err := net.Listen("unix", m.socketPath)
	syscall.Umask(old)

	if err != nil {
		return nil, nil, fmt.Errorf("create socket listener: %w", err)
	}

	if err := os.Chmod(m.socketPath, 0o600); err != nil {
		listener.Close()
		_ = os.Remove(m.socketPath)
		return nil, nil, fmt.Errorf("securing socket: %w", err)
	}

	m.server = grpc.NewServer()

	stopCh := make(chan struct{}, 1)
	RegisterDaemonServiceServer(m.server, &DaemonServer{
		downCh:    stopCh,
		startTime: time.Now(),
	})

	errCh := make(chan error, 1)
	go func() {
		err := m.server.Serve(listener)
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- err
		}
	}()

	return stopCh, errCh, nil
}

func (m *SockerManager) StopSignal() {
	if m.server != nil {
		m.server.GracefulStop()
	}
}
