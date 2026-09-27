package task

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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

func (c *Client) InvokeTask(
	ctx context.Context,
	command string,
	args []string,
	name *string,
	wd string,
	envs map[string]string,
) (*Task, error) {
	in := &InvokeTaskRequest{
		Command:    command,
		Args:       args,
		Name:       name,
		WorkingDir: wd,
		EnvVars:    envs,
	}
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

func (c *Client) StopTask(ctx context.Context, taskName string) (string, *Task, error) {
	in := &StopTaskRequest{Name: taskName}
	resp, err := c.remote.StopTask(ctx, in)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return "", nil, fmt.Errorf("connection or transport: %w", err)
		}

		switch st.Code() {
		case codes.NotFound:
			return "", nil, fmt.Errorf("task does not exist: %s", st.Message())
		case codes.FailedPrecondition:
			return "", nil, fmt.Errorf("it cannot be stopped; it has already finished: %s", st.Message())
		default:
			return "", nil, fmt.Errorf("unexpected error: %s, %s", st.Code(), st.Message())
		}
	}

	return resp.GetVia(), resp.GetTask(), nil
}
