FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy shared pkg first for caching
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY tenant-auth/go.mod ./tenant-auth/

# Initialize workspace inside container
WORKDIR /app
RUN go work init ./tenant-auth ./pkg
RUN go mod download

# Copy source
COPY pkg ./pkg
COPY tenant-auth ./tenant-auth

WORKDIR /app/tenant-auth
RUN go build -o /bin/tenant-auth ./cmd/server

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /bin/tenant-auth /bin/tenant-auth

# Add curl for healthcheck
RUN apk add --no-cache curl

CMD ["/bin/tenant-auth"]
