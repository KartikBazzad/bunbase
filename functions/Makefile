# Makefile for BunBase Functions

.PHONY: all build build-quickjs-worker clean test

all: build

build: build-quickjs-worker
	@echo "Building Go functions server..."
	go build -o functions ./cmd/functions

build-quickjs-worker:
	@echo "Building QuickJS-NG worker..."
	@if [ -d "cmd/quickjs-worker" ]; then \
		cd cmd/quickjs-worker && $(MAKE); \
	else \
		echo "QuickJS worker directory not found, skipping..."; \
	fi

clean:
	@echo "Cleaning..."
	rm -f functions
	@if [ -d "cmd/quickjs-worker" ]; then \
		cd cmd/quickjs-worker && $(MAKE) clean; \
	fi

test:
	@echo "Running tests..."
	go test ./...

install-deps:
	@echo "Installing dependencies..."
	@echo "Note: QuickJS-NG must be cloned and built separately"
	@echo "  cd .. && git clone https://github.com/quickjs-ng/quickjs.git quickjs-ng"
	@echo "  cd quickjs-ng && make"
