GOFLAGS ?= -ldflags="-s -w"

# Build targets
./bin/api:
	go build $(GOFLAGS) -o ./bin/api ./cmd/api

./bin/dashboard:
	go build $(GOFLAGS) -o ./bin/dashboard ./cmd/dashboard

./bin/operion:
	go build $(GOFLAGS) -o ./bin/operion ./cmd/operion

./bin/api-linux-amd64: ./cmd/api
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o ./bin/api-linux-amd64 ./cmd/api

# Main build commands
.PHONY: build build-linux clean test test-coverage test-integration fmt lint docs mod-check

build: ./bin/api ./bin/dashboard ./bin/operion

build-linux: ./bin/api-linux-amd64

clean:
	rm -rf ./bin

# Testing
test:
	go test ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-integration:
	go test -tags=integration ./...

# Code quality
fmt:
	go fmt ./...

lint:
	golangci-lint run

# Documentation
docs:
	godoc -http=:6060

# Dependencies
mod-check:
	go mod verify
	go mod tidy