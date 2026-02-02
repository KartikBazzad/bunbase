.PHONY: build test test-loadtest lint run docker-build docker-run loadtest clean

BINARY := bunkms
LOADTEST := bunkms-loadtest
GOFLAGS ?=

build:
	go build $(GOFLAGS) -o $(BINARY) ./cmd/server

loadtest:
	go build $(GOFLAGS) -o $(LOADTEST) ./cmd/loadtest

test:
	go test $(GOFLAGS) ./...

test-race:
	go test -race -short ./...

lint:
	go vet ./...
	@which golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || true

run: build
	./$(BINARY)

docker-build:
	docker build -f Dockerfile -t bunkms:latest ..

docker-run:
	docker run --rm -p 8080:8080 -e BUNKMS_MASTER_KEY=base64:$(shell echo -n "0123456789abcdef0123456789abcdef" | base64) bunkms:latest

test-loadtest:
	go test $(GOFLAGS) ./internal/loadtest/... -v -count=1

clean:
	rm -f $(BINARY) $(LOADTEST)
