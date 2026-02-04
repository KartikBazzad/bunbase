FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy shared pkg and module configurations
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY bun-auth/go.mod ./bun-auth/
COPY tenant-auth/go.mod ./tenant-auth/
COPY platform/go.mod platform/go.sum ./platform/
COPY bundoc-server/go.mod ./bundoc-server/
COPY bundoc/go.mod ./bundoc/
COPY bunder/go.mod ./bunder/
COPY bunder-manager/go.mod ./bunder-manager/
COPY bun-kms/go.mod ./bun-kms/
COPY buncast/go.mod ./buncast/
COPY functions/go.mod ./functions/
COPY cmd/bundoc-bench/go.mod ./cmd/bundoc-bench/

# Initialize workspace
WORKDIR /app
RUN go work init ./platform ./pkg ./bun-auth ./tenant-auth ./bundoc-server ./bundoc ./bunder ./bunder-manager ./bun-kms ./buncast ./functions ./cmd/bundoc-bench
RUN go mod download

# Copy source code relative to workspace root (assuming build context is root)
COPY pkg ./pkg
COPY platform ./platform
COPY bun-auth ./bun-auth
COPY tenant-auth ./tenant-auth
COPY bundoc-server ./bundoc-server
COPY bundoc ./bundoc
COPY bunder ./bunder
COPY bunder-manager ./bunder-manager
COPY bun-kms ./bun-kms
COPY buncast ./buncast
COPY functions ./functions
COPY cmd/bundoc-bench ./cmd/bundoc-bench

WORKDIR /app/platform
RUN go build -o /bin/platform-server ./cmd/server


FROM oven/bun:1.0-alpine as bun-base
# We use a multi-stage build to get bun binary or just use oven/bun as base?
# The platform binary is Go.
# We can use debian-slim and install bun?
# Or use oven/bun:debian as base and copy the go binary?
# Let's use `oven/bun:1.1-debian` (or just oven/bun) as the runtime image base.
# It has `bun` installed.

FROM oven/bun:1.1-slim as runtime
WORKDIR /app
COPY --from=builder /bin/platform-server /bin/platform-server

# Add curl for healthcheck and migrations (and postgresql-client)
RUN apt-get update && apt-get install -y curl postgresql-client && rm -rf /var/lib/apt/lists/*

# Create migrations directory
COPY platform/migrations /app/migrations

# Enable server-side builder
COPY platform/internal/builder /app/internal/builder

ENV PLATFORM_BUILDER_SCRIPT=/app/internal/builder/builder.ts

CMD ["/bin/platform-server", "-port", "3001"]
