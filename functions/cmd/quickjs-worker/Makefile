# Makefile for QuickJS-NG Worker
# 
# This builds the quickjs-worker binary that embeds QuickJS-NG and libuv

CC = gcc
CFLAGS = -O2 -Wall -fPIC -std=c99
LDFLAGS = 

# QuickJS-NG directory (assumes quickjs-ng is cloned in bunbase directory)
QUICKJS_NG_DIR ?= ../../../quickjs-ng
QUICKJS_NG_INCLUDE = $(QUICKJS_NG_DIR)
QUICKJS_NG_LIB = $(QUICKJS_NG_DIR)/build

# libuv (system package or custom path)
LIBUV_CFLAGS ?= $(shell pkg-config --cflags libuv 2>/dev/null || echo "")
LIBUV_LIBS ?= $(shell pkg-config --libs libuv 2>/dev/null || echo "-luv")

# Include directories
INCLUDES = -I$(QUICKJS_NG_INCLUDE) $(LIBUV_CFLAGS)

# Libraries
LIBS = -L$(QUICKJS_NG_LIB) -lqjs $(LIBUV_LIBS) -lm -ldl -lpthread

# Source files
SOURCES = main.c $(QUICKJS_NG_DIR)/quickjs-libc.c
OBJECTS = main.o quickjs-libc.o

# Target
TARGET = quickjs-worker

.PHONY: all clean check-deps

all: check-deps $(TARGET)

$(TARGET): $(OBJECTS)
	$(CC) $(CFLAGS) $(LDFLAGS) -o $(TARGET) $(OBJECTS) $(LIBS)

main.o: main.c
	$(CC) $(CFLAGS) $(INCLUDES) -c $< -o $@

quickjs-libc.o: $(QUICKJS_NG_DIR)/quickjs-libc.c
	$(CC) $(CFLAGS) $(INCLUDES) -c $< -o $@

check-deps:
	@echo "Checking dependencies..."
	@if [ ! -d "$(QUICKJS_NG_DIR)" ]; then \
		echo "ERROR: QuickJS-NG directory not found at $(QUICKJS_NG_DIR)"; \
		echo "Please clone QuickJS-NG: git clone https://github.com/quickjs-ng/quickjs.git $(QUICKJS_NG_DIR)"; \
		echo "Then build it: cd $(QUICKJS_NG_DIR) && make"; \
		exit 1; \
	fi
	@if [ ! -f "$(QUICKJS_NG_LIB)/libqjs.a" ]; then \
		echo "WARNING: QuickJS-NG library not found. Building..."; \
		cd $(QUICKJS_NG_DIR) && make; \
	fi
	@echo "Dependencies OK"

clean:
	rm -f $(OBJECTS) $(TARGET)

install: $(TARGET)
	cp $(TARGET) /usr/local/bin/ || cp $(TARGET) ../
