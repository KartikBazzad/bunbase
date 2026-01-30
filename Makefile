DOCDB_DIR := docdb
DOCDB_BIN := $(DOCDB_DIR)/docdb
SHELL_BIN := $(DOCDB_DIR)/docdbsh

FUNCTIONS_DIR := functions
PLATFORM_DIR := platform
PLATFORM_WEB_DIR := platform-web

.PHONY: all docdb shell functions platform platform-web dev clean matrix-test matrix-test-quick matrix-test-full matrix-analyze matrix-clean docdb-server docdb-server-multi docdb-server-custom docdb-server-debug

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

## Matrix Testing Commands

# Default socket and directories
SOCKET := /tmp/docdb.sock
WAL_DIR := ./docdb/data/wal
RESULTS_DIR := ./matrix_results

## Run quick matrix test (1,10,20 DBs, 1,20,50 conns, 1 worker, 1min duration)
matrix-test-quick:
	@echo "Running quick matrix test..."
	go run ./docdb/tests/load/cmd/matrix_runner/main.go \
		-databases "1,10,20" \
		-connections "1,20,50" \
		-workers "1" \
		-duration 1m \
		-socket $(SOCKET) \
		-wal-dir $(WAL_DIR) \
		-output-dir $(RESULTS_DIR)

## Run full matrix test (1,3,6,12 DBs, 1,5,10,20 conns, 1,2,5,10 workers, 5min duration)
matrix-test-full:
	@echo "Running full matrix test (this will take a while)..."
	go run ./docdb/tests/load/cmd/matrix_runner/main.go \
		-databases "1,3,6,12" \
		-connections "1,5,10,20" \
		-workers "1,2,5,10" \
		-duration 5m \
		-socket $(SOCKET) \
		-wal-dir $(WAL_DIR) \
		-output-dir $(RESULTS_DIR)

## Run custom matrix test (use: make matrix-test DB="1,10" CONN="1,20" WORK="1" DUR="1m")
matrix-test:
	@echo "Running matrix test with custom parameters..."
	@if [ -z "$(DB)" ] || [ -z "$(CONN)" ] || [ -z "$(WORK)" ]; then \
		echo "Usage: make matrix-test DB=\"1,10\" CONN=\"1,20\" WORK=\"1\" DUR=\"1m\""; \
		exit 1; \
	fi
	go run ./docdb/tests/load/cmd/matrix_runner/main.go \
		-databases "$(DB)" \
		-connections "$(CONN)" \
		-workers "$(WORK)" \
		-duration $(if $(DUR),$(DUR),1m) \
		-socket $(SOCKET) \
		-wal-dir $(WAL_DIR) \
		-output-dir $(RESULTS_DIR)

## Analyze matrix test results (use: make matrix-analyze OUTPUT="analysis.md")
matrix-analyze:
	@echo "Analyzing matrix test results..."
	@OUTPUT_FILE=$(if $(OUTPUT),$(OUTPUT),analysis.md); \
	go run ./docdb/tests/load/cmd/analyze_matrix/main.go \
		-results-dir $(RESULTS_DIR) \
		-output $(RESULTS_DIR)/reports/$$OUTPUT_FILE

## Clean matrix test results
matrix-clean:
	@echo "Cleaning matrix test results..."
	rm -rf $(RESULTS_DIR)/json/*.json
	rm -rf $(RESULTS_DIR)/csv_dbs/*
	rm -rf $(RESULTS_DIR)/csv_global/*
	rm -rf $(RESULTS_DIR)/reports/*.md
	rm -f $(RESULTS_DIR)/summary.txt

## Start DocDB server (default: single worker, no multi-writer)
docdb-server:
	@echo "Starting DocDB server (single worker)..."
	@echo "Socket: $(SOCKET)"
	@echo "Data dir: ./docdb/data"
	./$(DOCDB_BIN) \
		-data-dir ./docdb/data \
		-socket $(SOCKET)

## Start DocDB server with multi-worker scheduler (4 workers, max 16)
docdb-server-multi:
	@echo "Starting DocDB server (multi-worker: 4 workers, max 16)..."
	@echo "Socket: $(SOCKET)"
	@echo "Data dir: ./docdb/data"
	./$(DOCDB_BIN) \
		-unsafe-multi-writer \
		-sched-workers 4 \
		-sched-max-workers 16 \
		-data-dir ./docdb/data \
		-socket $(SOCKET)

## Start DocDB server with custom scheduler config (use: make docdb-server-custom WORKERS=8 MAX=32)
docdb-server-custom:
	@echo "Starting DocDB server (custom: $(if $(WORKERS),$(WORKERS),4) workers, max $(if $(MAX),$(MAX),16))..."
	@echo "Socket: $(SOCKET)"
	@echo "Data dir: ./docdb/data"
	./$(DOCDB_BIN) \
		-unsafe-multi-writer \
		-sched-workers $(if $(WORKERS),$(WORKERS),4) \
		-sched-max-workers $(if $(MAX),$(MAX),16) \
		-data-dir ./docdb/data \
		-socket $(SOCKET)

## Start DocDB server with debug mode
docdb-server-debug:
	@echo "Starting DocDB server (debug mode)..."
	@echo "Socket: $(SOCKET)"
	@echo "Data dir: ./docdb/data"
	./$(DOCDB_BIN) \
		-debug \
		-data-dir ./docdb/data \
		-socket $(SOCKET)
