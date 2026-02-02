FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod/sum files
COPY bunder/go.mod ./bunder/
COPY buncast/go.mod ./buncast/

# Copy modules
COPY bunder ./bunder
COPY buncast ./buncast

# Download dependencies
WORKDIR /app/bunder
RUN go mod download

# Build
RUN go build -o /bin/bunder ./cmd/server

# Final Stage
FROM alpine:3.18

WORKDIR /app
COPY --from=builder /bin/bunder /bin/bunder

RUN apk add --no-cache curl

# Data directory
RUN mkdir -p /data
VOLUME /data

# Ports
EXPOSE 6379 8080

# Run
CMD ["/bin/bunder", "-addr", ":6379", "-http", ":8080", "-data", "/data"]
