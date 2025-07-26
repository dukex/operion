GOFLAGS ?= -ldflags="-s -w"

./bin/operion-worker:
	go build $(GOFLAGS) -o ./bin/operion-worker ./cmd/operion-worker

./bin/operion-dispatcher:
	go build $(GOFLAGS) -o ./bin/operion-dispatcher ./cmd/operion-dispatcher

./bin/operion-api:
	go build $(GOFLAGS) -o ./bin/operion-api ./cmd/operion-api


.PHONY: build build-linux clean test test-coverage fmt lint docs mod-check

build: ./bin/operion-worker ./bin/operion-dispatcher ./bin/operion-api

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

examples-dispatcher:
	cd examples/ && docker compose up dispatcher-kafka -d --build

examples-workder:
	cd examples/ && docker compose up worker-kafka -d --build

examples-all: examples-stop examples-workder examples-dispatcher
	cd examples/ && docker compose up akhq -d
	open http://localhost:8080

examples-stop:
	cd examples/ && docker compose down

release:
	git tag ${tag} -m "Release ${tag}" -f
	git push origin ${tag} -f
	git push origin main
	docker buildx build -t dukex/operion:${tag} .
	docker tag dukex/operion:${tag} docker.io/dukex/operion:${tag}
	docker push docker.io/dukex/operion:${tag}
	docker tag dukex/operion:${tag} docker.io/dukex/operion:latest
	docker push docker.io/dukex/operion:latest