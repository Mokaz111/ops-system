.PHONY: run build test tidy fmt vet

BINARY=bin/server

run:
	go run ./cmd/server

build:
	mkdir -p bin
	go build -o $(BINARY) ./cmd/server

test:
	go test -race -cover ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...
