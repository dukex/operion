FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . ./

RUN go build -v -o /app/bin/operion-trigger /app/cmd/operion-trigger
RUN go build -v -o /app/bin/operion-worker /app/cmd/operion-worker

FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/bin/operion-trigger /bin/operion-trigger
COPY --from=builder /app/bin/operion-worker /bin/operion-worker

