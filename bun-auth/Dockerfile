FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy shared pkg first for caching
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY bun-auth/go.mod ./bun-auth/

# Initialize workspace inside container -> Use only what we duplicate
WORKDIR /app
RUN go work init ./bun-auth ./pkg
RUN go mod download

# Copy source
COPY pkg ./pkg
COPY bun-auth ./bun-auth

WORKDIR /app/bun-auth
RUN go build -o /bin/bun-auth ./cmd/server

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /bin/bun-auth /bin/bun-auth

# Add curl for healthcheck
RUN apk add --no-cache curl

CMD ["/bin/bun-auth"]
