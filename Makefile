.PHONY: run deps vet lint test bench build clean

BINARY_NAME=asyncker
BUILD_DIR=./bin

run:
	go run . daemon --up

deps:
	go mod download

vet:
	go vet ./...

lint:
	golangci-lint run

test: vet lint
	go test -race ./...

bench:
	go test -bench=. ./...

generate:
	go generate ./...

build: generate
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) .

clean:
	rm -rf $(BUILD_DIR)/
