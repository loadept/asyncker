package task

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
