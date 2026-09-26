package task

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Client struct {
	conn   *grpc.ClientConn
	remote TaskServiceClient
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
		remote: NewTaskServiceClient(conn),
	}), nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) InvokeTask(ctx context.Context, in *InvokeTaskRequest) (*Task, error) {
	resp, err := c.remote.InvokeTask(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("RPC InvokeTask: %w", err)
	}

	return resp.GetTask(), nil
}

func (c *Client) ListTasks(ctx context.Context) ([]*Task, error) {
	resp, err := c.remote.ListTasks(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("RPC ListTasks: %w", err)
	}

	return resp.GetTasks(), nil
}
