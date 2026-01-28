DOCDB_DIR := docdb
DOCDB_BIN := $(DOCDB_DIR)/docdb
SHELL_BIN := $(DOCDB_DIR)/docdbsh

FUNCTIONS_DIR := functions
PLATFORM_DIR := platform
PLATFORM_WEB_DIR := platform-web

.PHONY: all docdb shell functions platform platform-web dev clean

## Build all core backend components.
all: docdb shell functions platform

## Build the DocDB server binary (./docdb/docdb).
docdb:
	@echo "Building docdb server..."
	cd $(DOCDB_DIR) && go build -o ./docdb ./cmd/docdb

## Build the DocDB shell binary (./docdb/docdbsh).
shell:
	@echo "Building docdb shell..."
	cd $(DOCDB_DIR) && go build -o ./docdbsh ./cmd/docdbsh

## Build the Functions service binary (./functions/functions).
functions:
	@echo "Building functions service..."
	cd $(FUNCTIONS_DIR) && go build -o ./functions ./cmd/functions

## Build the Platform API server binary (./platform/platform-server).
platform:
	@echo "Building platform API server..."
	cd $(PLATFORM_DIR) && go build -o ./platform-server ./cmd/server

## Install dependencies and run the Platform Web dev server.
platform-web:
	@echo "Starting platform web dev server with Bun..."
	cd $(PLATFORM_WEB_DIR) && bun install && bun run dev

## Show recommended commands for running the full dev environment.
dev:
	@echo "BunBase dev environment:"
	@echo "  # Terminal 1: build and run DocDB"
	@echo "  cd $(DOCDB_DIR) && go build -o ./docdb ./cmd/docdb && ./docdb"
	@echo ""
	@echo "  # Terminal 2: run Functions service"
	@echo "  cd $(FUNCTIONS_DIR) && go build -o ./functions ./cmd/functions && ./functions --data-dir ./data --socket /tmp/functions.sock"
	@echo ""
	@echo "  # Terminal 3: run Platform API"
	@echo "  cd $(PLATFORM_DIR) && go build -o ./platform-server ./cmd/server && ./platform-server \\"
	@echo "    --db-path ./data/platform.db \\"
	@echo "    --port 3001 \\"
	@echo "    --functions-socket /tmp/functions.sock \\"
	@echo "    --bundle-path ../functions/data/bundles \\"
	@echo "    --cors-origin http://localhost:5173"
	@echo ""
	@echo "  # Terminal 4: run Platform Web with Bun"
	@echo "  cd $(PLATFORM_WEB_DIR) && bun install && bun run dev"

## Remove built docdb binaries.
clean:
	@echo "Cleaning docdb binaries..."
	rm -f $(DOCDB_BIN) $(SHELL_BIN)


