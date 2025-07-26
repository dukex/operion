FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY cmd ./cmd
COPY pkg ./pkg

RUN go build -ldflags="-s -w" -v -o /app/bin/operion-dispatcher /app/cmd/operion-dispatcher
RUN go build -ldflags="-s -w" -v -o /app/bin/operion-worker /app/cmd/operion-worker
RUN go build -ldflags="-s -w" -v -o /app/bin/operion-api /app/cmd/operion-api

FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/bin/operion-dispatcher /bin/operion-dispatcher
COPY --from=builder /app/bin/operion-worker /bin/operion-worker
COPY --from=builder /app/bin/operion-api /bin/operion-api
