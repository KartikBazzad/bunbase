FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy module files
COPY buncast/go.mod ./buncast/
COPY bunder/go.mod ./bunder/

# Initialize workspace
WORKDIR /app
RUN go work init ./buncast ./bunder
RUN go mod download

# Copy source code
COPY buncast ./buncast
COPY bunder ./bunder

WORKDIR /app/buncast
RUN go build -o /bin/buncast ./cmd/server

FROM alpine:3.18
WORKDIR /app

COPY --from=builder /bin/buncast /bin/buncast

# Add curl for healthcheck
RUN apk add --no-cache curl

CMD ["/bin/buncast", "-socket", "/tmp/buncast.sock", "-http", ":8081"]
