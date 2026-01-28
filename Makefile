DOCDB_DIR := docdb
DOCDB_BIN := $(DOCDB_DIR)/docdb
SHELL_BIN := $(DOCDB_DIR)/docdbsh

.PHONY: all docdb shell clean

## Build both the DocDB server binary and the interactive shell.
all: docdb shell

## Build the DocDB server binary (./docdb/docdb).
docdb:
	@echo "Building docdb server..."
	cd $(DOCDB_DIR) && go build -o ./docdb ./cmd/docdb

## Build the DocDB shell binary (./docdb/docdbsh).
shell:
	@echo "Building docdb shell..."
	cd $(DOCDB_DIR) && go build -o ./docdbsh ./cmd/docdbsh

## Remove built docdb binaries.
clean:
	@echo "Cleaning docdb binaries..."
	rm -f $(DOCDB_BIN) $(SHELL_BIN)

