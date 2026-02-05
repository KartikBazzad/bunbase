package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/kartikbazzad/bunbase/pkg/config"
	"github.com/kartikbazzad/bunbase/pkg/logger"
	"github.com/kartikbazzad/bunbase/tenant-auth/internal/api"
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
		os.Setenv("TENANTAUTH_BUNDOC_URL", "http://bundoc-server:8080")
	}
	if os.Getenv("TENANTAUTH_JWT_SECRET") == "" {
		os.Setenv("TENANTAUTH_JWT_SECRET", "tenant-dev-secret-key")
	}
	if os.Getenv("TENANTAUTH_BUNKMS_URL") == "" {
		os.Setenv("TENANTAUTH_BUNKMS_URL", "")
	}

	if err := config.Load("TENANTAUTH_", &cfg); err != nil {
		log.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 3. Connect to Database (Bundoc)
	database := db.NewBundocDB(cfg.Bundoc.URL)
	log.Info("Initialized Bundoc Client", "url", cfg.Bundoc.URL)
	// No explicit Close() generic method in struct, but NewBundocDB returns *BundocDB which has Close().
	defer database.Close()

	// 4. Optional KMS client for provider secrets
	var kmsClient *kms.Client
	if cfg.Bunkms.URL != "" {
		kmsClient = kms.NewClient(cfg.Bunkms.URL, cfg.Bunkms.Token)
		log.Info("KMS client enabled for provider secrets", "url", cfg.Bunkms.URL)
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
