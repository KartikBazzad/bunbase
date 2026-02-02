# Build stage. Build context: repo root so bunder is available (replace => ../bunder).
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY bunder/ ./bunder/
COPY bun-kms/ ./bun-kms/
WORKDIR /app/bun-kms
RUN go mod download && CGO_ENABLED=0 go build -ldflags="-s -w" -o bunkms ./cmd/server

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/bun-kms/bunkms .
EXPOSE 8080
ENTRYPOINT ["./bunkms"]
