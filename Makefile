GOFLAGS ?= -ldflags="-s -w"

./bin/api:
	go build $(GOFLAGS) -o ./bin/api ./cmd/api

./bin/dashboard:
	go build $(GOFLAGS) -o ./bin/dashboard ./cmd/dashboard

./bin/operion-worker:
	go build $(GOFLAGS) -o ./bin/operion-worker ./cmd/operion-worker

./bin/api-linux-amd64: ./cmd/api
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o ./bin/api-linux-amd64 ./cmd/api

.PHONY: build build-linux clean test test-coverage test-integration fmt lint docs mod-check

build: ./bin/api ./bin/dashboard ./bin/operion-worker

build-linux: ./bin/api-linux-amd64

clean:
	rm -rf ./bin

test:
	go test ./...

test-integration:
	go test -v ./internal/adapters/web/

test-all: test test-integration

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-integration:
	go test -tags=integration ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

docs:
	godoc -http=:6060

mod-check:
	go mod verify
	go mod tidy