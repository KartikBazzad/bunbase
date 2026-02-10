# Multi-stage: build bunder-manager (uses embedded Bunder instances)
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go.mod files for dependency resolution
COPY bunder/go.mod ./bunder/
COPY buncast/go.mod ./buncast/
COPY bunder-manager/go.mod ./bunder-manager/

# Copy source code
COPY bunder ./bunder
COPY buncast ./buncast
COPY bunder-manager ./bunder-manager

# Build bunder-manager
WORKDIR /app/bunder-manager
RUN go mod download && go build -o /bin/bunder-manager ./cmd/server

# Final stage
FROM alpine:3.18

RUN apk add --no-cache curl

COPY --from=builder /bin/bunder-manager /bin/bunder-manager

RUN mkdir -p /data
VOLUME /data

EXPOSE 8080

# Manager listens on 8080; uses embedded Bunder instances (no separate processes)
CMD ["/bin/bunder-manager", "-addr", ":8080", "-data", "/data"]
