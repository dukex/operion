GOOS ?= linux
GOARCH ?= amd64
GOFLAGS ?= -ldflags="-s -w"

./build/cmd/api:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) -o ./build/cmd/api ./cmd/api

.PHONY: build
.PHONY: clean

build: ./build/cmd/api
	
clean:
	rm -rf ./build/cmd