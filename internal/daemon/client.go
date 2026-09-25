package daemon

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type Client struct {
	conn   *grpc.ClientConn
	remote DaemonServiceClient
}

func NewClient(socketPath string) (*Client, error) {
	target := "unix://" + socketPath
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to socket: %w", err)
	}

	return new(Client{
		conn:   conn,
		remote: NewDaemonServiceClient(conn),
	}), nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) StopDaemon(ctx context.Context) error {
	_, err := c.remote.StopDaemon(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("RPC StopDaemon: %w", err)
	}
	return nil
}

func (c *Client) DaemonStatus(ctx context.Context) (*StatusResponse, error) {
	res, err := c.remote.DaemonStatus(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("RPC DaemonStatus: %w", err)
	}
	return res, nil
}
