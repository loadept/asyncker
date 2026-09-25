package daemon

import (
	context "context"
	"log"
	"os"
	"time"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type DaemonServer struct {
	UnimplementedDaemonServiceServer
	downCh    chan struct{}
	startTime time.Time
}

func (s *DaemonServer) StopDaemon(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	log.Println("Cositas")
	select {
	case s.downCh <- struct{}{}:
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
