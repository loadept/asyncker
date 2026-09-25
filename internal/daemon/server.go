package daemon

import (
	context "context"
	"log/slog"
	"os"
	"time"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type DaemonServer struct {
	UnimplementedDaemonServiceServer
	logger    *slog.Logger
	downCh    chan<- struct{}
	startTime time.Time
}

func NewDaemonServer(logger *slog.Logger, downCh chan<- struct{}) *DaemonServer {
	return new(DaemonServer{logger: logger, downCh: downCh, startTime: time.Now()})
}

func (s *DaemonServer) StopDaemon(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	select {
	case s.downCh <- struct{}{}:
		s.logger.Info("RPC call received to stop the server")
	default:
	}
	return &emptypb.Empty{}, nil
}

func (s *DaemonServer) DaemonStatus(ctx context.Context, req *emptypb.Empty) (*StatusResponse, error) {
	uptime := time.Since(s.startTime).String()

	return &StatusResponse{
		Running: true,
		Pid:     int32(os.Getpid()),
		Uptime:  uptime,
	}, nil
}
