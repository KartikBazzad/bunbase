package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kartikbazzad/bunbase/buncast/internal/broker"
	"github.com/kartikbazzad/bunbase/buncast/internal/config"
	httpsrv "github.com/kartikbazzad/bunbase/buncast/internal/http"
	"github.com/kartikbazzad/bunbase/buncast/internal/ipc"
	"github.com/kartikbazzad/bunbase/buncast/internal/logger"
)

func main() {
	cfgPath := flag.String("config", "", "Path to config file (optional)")
	socketPath := flag.String("socket", "/tmp/buncast.sock", "Unix socket path")
	httpAddr := flag.String("http", ":8081", "HTTP listen address (empty to disable)")
	debugMode := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	cfg := config.DefaultConfig()
	cfg.IPC.SocketPath = *socketPath
	cfg.IPC.DebugMode = *debugMode
	cfg.HTTP.ListenAddr = *httpAddr
	cfg.HTTP.Enabled = *httpAddr != ""

	if cfgPath != nil && *cfgPath != "" {
		fmt.Printf("Config file not yet implemented, using defaults\n")
	}

	logr := logger.Default()
	logr.Info("Starting Buncast...")
	logr.Info("Socket: %s", cfg.IPC.SocketPath)
	if cfg.HTTP.Enabled {
		logr.Info("HTTP: %s", cfg.HTTP.ListenAddr)
	}
	if cfg.IPC.DebugMode {
		logr.Info("Debug mode: enabled")
	}

	b := broker.New(256)
	ipcServer, err := ipc.NewServer(cfg, logr, b)
	if err != nil {
		log.Fatalf("Failed to create IPC server: %v", err)
	}
	if err := ipcServer.Start(); err != nil {
		log.Fatalf("Failed to start IPC server: %v", err)
	}

	var srv *httpsrv.Server
	if cfg.HTTP.Enabled {
		srv = httpsrv.NewServer(cfg, logr, b)
		go func() {
			if err := srv.Start(); err != nil && err != http.ErrServerClosed {
				logr.Error("HTTP server: %v", err)
			}
		}()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logr.Info("Shutting down...")
	if srv != nil {
		_ = srv.Stop()
	}
	if err := ipcServer.Stop(); err != nil {
		logr.Error("Error during IPC shutdown: %v", err)
	}
	logr.Info("Buncast stopped")
	os.Exit(0)
}
