package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/kartikbazzad/bunbase/pkg/config"
	"github.com/kartikbazzad/bunbase/pkg/logger"
	"github.com/kartikbazzad/bunbase/tenant-auth/internal/api"
	tenantconfig "github.com/kartikbazzad/bunbase/tenant-auth/internal/config"
	"github.com/kartikbazzad/bunbase/tenant-auth/internal/db"
	"github.com/kartikbazzad/bunbase/tenant-auth/internal/kms"
)

// AppConfig holds the service configuration
type AppConfig struct {
	Port   int `mapstructure:"port"`
	Bundoc struct {
		URL string `mapstructure:"url"`
	} `mapstructure:"bundoc"`
	JWT struct {
		Secret string `mapstructure:"secret"`
	} `mapstructure:"jwt"`
	Bunkms struct {
		URL   string `mapstructure:"url"`
		Token string `mapstructure:"token"`
	} `mapstructure:"bunkms"`
}

func main() {
	// 1. Initialize Logger
	logger.Init(logger.Config{Level: "DEBUG", Format: "json"})
	log := logger.Get()
	log.Info("Starting TenantAuth Service...")

	// 2. Load Config
	var cfg AppConfig

	// Set defaults
	if os.Getenv("TENANTAUTH_PORT") == "" {
		os.Setenv("TENANTAUTH_PORT", "8083")
	}
	if os.Getenv("TENANTAUTH_BUNDOC_URL") == "" {
		os.Setenv("TENANTAUTH_BUNDOC_URL", "http://bundoc-auth:8080")
	}
	if os.Getenv("TENANTAUTH_BUNKMS_URL") == "" {
		os.Setenv("TENANTAUTH_BUNKMS_URL", "")
	}

	// Validate JWT secret (required, >= 32 bytes)
	jwtSecret, err := tenantconfig.ValidateJWTSecret()
	if err != nil {
		log.Error("JWT secret validation failed", "error", err)
		os.Exit(1)
	}
	os.Setenv("TENANTAUTH_JWT_SECRET", jwtSecret)

	if err := config.Load("TENANTAUTH_", &cfg); err != nil {
		log.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 3. Connect to Database (Bundoc) – use RPC when configured for lower latency
	var database db.DBClient
	if rpcAddr := os.Getenv("TENANTAUTH_BUNDOC_RPC_ADDR"); rpcAddr != "" {
		rpcDB := db.NewBundocRPCDB(rpcAddr)
		defer rpcDB.Close()
		database = rpcDB
		log.Info("Initialized Bundoc RPC client", "addr", rpcAddr)
	} else {
		httpDB := db.NewBundocDB(cfg.Bundoc.URL)
		defer httpDB.Close()
		database = httpDB
		log.Info("Initialized Bundoc HTTP client", "url", cfg.Bundoc.URL)
	}

	// 4. Optional KMS client for provider secrets (RPC preferred when set)
	var kmsClient kms.ClientInterface
	if rpcAddr := os.Getenv("TENANTAUTH_BUNKMS_RPC_ADDR"); rpcAddr != "" {
		rpcCl := kms.NewRPCClient(rpcAddr)
		defer rpcCl.Close()
		kmsClient = rpcCl
		log.Info("KMS RPC client enabled for provider secrets", "addr", rpcAddr)
	} else if cfg.Bunkms.URL != "" {
		kmsClient = kms.NewClient(cfg.Bunkms.URL, cfg.Bunkms.Token)
		log.Info("KMS HTTP client enabled for provider secrets", "url", cfg.Bunkms.URL)
	}

	// 5. Initialize Handler
	handler := api.NewHandler(database, cfg.JWT.Secret, kmsClient)

	// 6. Start HTTP Server
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Info("Listening on HTTP", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
