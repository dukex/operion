GOFLAGS ?= -ldflags="-s -w"

./bin/operion-worker:
	go build $(GOFLAGS) -o ./bin/operion-worker ./cmd/operion-worker

./bin/operion-trigger:
	go build $(GOFLAGS) -o ./bin/operion-trigger ./cmd/operion-trigger

.PHONY: build build-linux clean test test-coverage fmt lint docs mod-check

build: ./bin/operion-worker ./bin/operion-trigger

build-linux: build

clean:
	rm -rf ./bin

test:
	go test ./...

test-all: test

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	go fmt ./...

lint:
	golangci-lint run

docs:
	godoc -http=:6060

mod-check:
	go mod verify
	go mod tidy