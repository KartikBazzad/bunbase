FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy shared pkg and module configurations
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY bundoc/go.mod ./bundoc/
COPY buncast/go.mod ./buncast/
COPY bundoc-server/go.mod ./bundoc-server/

# Initialize workspace
WORKDIR /app
RUN go work init ./bundoc-server ./bundoc ./buncast ./pkg
RUN go mod download

# Copy source code
COPY pkg ./pkg
COPY bundoc ./bundoc
COPY buncast ./buncast
COPY bundoc-server ./bundoc-server

WORKDIR /app/bundoc-server
RUN go build -o /bin/bundoc-server .

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /bin/bundoc-server /bin/bundoc-server

# Add curl for healthcheck
RUN apk add --no-cache curl

# Create data directory
RUN mkdir -p /app/data

CMD ["/bin/bundoc-server", "-port", "4321"]
