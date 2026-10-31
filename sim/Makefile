.PHONY: build test lint

build:
	go build -o bin/swarmsim ./cmd/swarmsim

test:
	go test ./...

lint:
	golangci-lint run ./...

