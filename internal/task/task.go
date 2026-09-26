// Package task provides functionality for managing tasks by exposing gRPC
// functions to interact with the asyncker daemon
package task

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative task.proto
