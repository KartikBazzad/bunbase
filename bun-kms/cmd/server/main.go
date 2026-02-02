package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kartikbazzad/bunbase/bun-kms/internal/api"
	"github.com/kartikbazzad/bunbase/bun-kms/internal/audit"
	"github.com/kartikbazzad/bunbase/bun-kms/internal/core"
	"github.com/kartikbazzad/bunbase/bun-kms/internal/health"
	"github.com/kartikbazzad/bunbase/bun-kms/internal/metrics"
	"github.com/kartikbazzad/bunbase/bun-kms/internal/storage"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	logger := log.New(os.Stdout, "bun-kms ", log.LstdFlags)

	addr := os.Getenv("BUNKMS_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	masterKey, err := core.ParseMasterKey(os.Getenv("BUNKMS_MASTER_KEY"))
	if err != nil {
		logger.Fatalf("failed to load master key: %v", err)
	}

	var auditLog audit.Store
	if path := os.Getenv("BUNKMS_AUDIT_LOG"); path != "" {
		al, err := audit.NewLogger(path)
		if err != nil {
			logger.Fatalf("failed to open audit log: %v", err)
		}
		auditLog = al
		logger.Printf("audit log at %s", path)
	}

	var vault *core.Vault
	var secrets *core.SecretStore
	var store storage.Store
	dataPath := os.Getenv("BUNKMS_DATA_PATH")
	if dataPath != "" {
		store, err = storage.NewBunderStore(dataPath, masterKey, nil)
		if err != nil {
			if auditLog != nil {
				_ = auditLog.Close()
			}
			logger.Fatalf("failed to open bunder store: %v", err)
		}
		vault, err = core.NewVaultWithStore(store, auditLog)
		if err != nil {
			_ = store.Close()
			if auditLog != nil {
				_ = auditLog.Close()
			}
			logger.Fatalf("failed to init vault: %v", err)
		}
		secrets, err = core.NewSecretStoreWithStore(masterKey, store, auditLog)
		if err != nil {
			_ = store.Close()
			if auditLog != nil {
				_ = auditLog.Close()
			}
			logger.Fatalf("failed to init secret store: %v", err)
		}
		logger.Printf("persistence enabled via bunder at %s", dataPath)
	} else {
		vault, err = core.NewVaultWithStore(nil, auditLog)
		if err != nil {
			if auditLog != nil {
				_ = auditLog.Close()
			}
			logger.Fatalf("failed to init vault: %v", err)
		}
		secrets, err = core.NewSecretStoreWithStore(masterKey, nil, auditLog)
		if err != nil {
			if auditLog != nil {
				_ = auditLog.Close()
			}
			logger.Fatalf("failed to init secret store: %v", err)
		}
	}

	jwtSecret := []byte(os.Getenv("BUNKMS_JWT_SECRET"))
	server := api.NewServer(vault, secrets, logger, jwtSecret)
	var storeCheck func() error
	if store != nil {
		storeCheck = func() error { return nil }
	}
	mux := http.NewServeMux()
	mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		health.Handler(true, storeCheck).ServeHTTP(w, r)
	}))
	mux.Handle("/ready", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		health.NewReadiness(storeCheck).ServeHTTP(w, r)
	}))
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", metrics.Middleware(server.Handler()))

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-quit
		logger.Print("shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
	}()

	logger.Printf("listening on %s", addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("server error: %v", err)
	}
	if store != nil {
		if err := store.Close(); err != nil {
			logger.Printf("store close: %v", err)
		}
	}
	if auditLog != nil {
		if err := auditLog.Close(); err != nil {
			logger.Printf("audit log close: %v", err)
		}
	}
	logger.Print("server stopped")
}
