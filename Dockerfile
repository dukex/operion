FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY cmd ./cmd
COPY pkg ./pkg

RUN go build -ldflags="-s -w" -v -o /app/bin/operion-worker /app/cmd/operion-worker
RUN go build -ldflags="-s -w" -v -o /app/bin/operion-api /app/cmd/operion-api
RUN go build -ldflags="-s -w" -v -o /app/bin/operion-activator /app/cmd/operion-activator
RUN go build -ldflags="-s -w" -v -o /app/bin/operion-source-manager /app/cmd/operion-source-manager

FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/bin/operion-worker /bin/operion-worker
COPY --from=builder /app/bin/operion-api /bin/operion-api
COPY --from=builder /app/bin/operion-activator /bin/operion-activator
COPY --from=builder /app/bin/operion-source-manager /bin/operion-source-manager
