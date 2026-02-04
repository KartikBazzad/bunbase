FROM golang:1.24-alpine AS builder

# Install C build dependencies for QuickJS
RUN apk add --no-cache gcc make musl-dev git libuv-dev cmake

WORKDIR /app

# Copy shared pkg and module configurations for Workspace
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY functions/go.mod ./functions/

# Init workspace
WORKDIR /app
RUN go work init ./functions ./pkg
RUN go mod download

# Copy source code
COPY pkg ./pkg
COPY functions ./functions
COPY quickjs-ng ./quickjs-ng

# Build QuickJS-NG
WORKDIR /app/quickjs-ng
RUN rm -rf build && mkdir -p build && cd build && cmake .. && make

# Build QuickJS Worker
WORKDIR /app/functions/cmd/quickjs-worker
# Makefile expects quickjs-ng at ../../../quickjs-ng, which maps to /app/quickjs-ng
RUN make clean all

# Build Functions Service
WORKDIR /app/functions
RUN go build -o /bin/functions ./cmd/functions

# Runtime Stage
FROM alpine:3.18
WORKDIR /app

# Runtime deps
RUN apk add --no-cache curl ca-certificates libgcc

# Copy binaries
COPY --from=builder /bin/functions /bin/functions
COPY --from=builder /app/functions/cmd/quickjs-worker/quickjs-worker /bin/quickjs-worker

# Copy worker/init scripts
COPY functions/worker /app/worker

# Create data directory
RUN mkdir -p /app/data

# Defaults
ENV FUNCTIONS_HTTP_PORT=8080
ENV WORKER_BINARY_PATH=/bin/quickjs-worker
ENV WORKER_SCRIPT_PATH=/app/worker/worker.ts
ENV INIT_SCRIPT_PATH=/app/worker/init.js

CMD ["/bin/functions"]
