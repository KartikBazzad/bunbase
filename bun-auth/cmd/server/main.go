package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/kartikbazzad/bunbase/bun-auth/internal/api"
	"github.com/kartikbazzad/bunbase/bun-auth/internal/db"
	"github.com/kartikbazzad/bunbase/pkg/config"
	"github.com/kartikbazzad/bunbase/pkg/logger"
)

// AppConfig holds the service configuration
type AppConfig struct {
	Port int `mapstructure:"port"`
	DB   struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Name     string `mapstructure:"name"`
	} `mapstructure:"db"`
	JWT struct {
		PrivateKeyPath string `mapstructure:"private_key_path"`
		PublicKeyPath  string `mapstructure:"public_key_path"`
	} `mapstructure:"jwt"`
}

func main() {
	// 1. Initialize Logger
	logger.Init(logger.Config{Level: "DEBUG", Format: "json"})
	log := logger.Get()
	log.Info("Starting BunAuth Service...")

	// 2. Load Config
	var cfg AppConfig
	// Set defaults
	// Set defaults only if not set
	if os.Getenv("BUNAUTH_PORT") == "" {
		os.Setenv("BUNAUTH_PORT", "8081")
	}
	if os.Getenv("BUNAUTH_DB_HOST") == "" {
		os.Setenv("BUNAUTH_DB_HOST", "localhost")
	}
	if os.Getenv("BUNAUTH_DB_PORT") == "" {
		os.Setenv("BUNAUTH_DB_PORT", "5432")
	}
	if os.Getenv("BUNAUTH_DB_USER") == "" {
		os.Setenv("BUNAUTH_DB_USER", "bunadmin")
	}
	if os.Getenv("BUNAUTH_DB_PASSWORD") == "" {
		os.Setenv("BUNAUTH_DB_PASSWORD", "bunpassword")
	}
	if os.Getenv("BUNAUTH_DB_NAME") == "" {
		os.Setenv("BUNAUTH_DB_NAME", "bunbase_system")
	}

	if err := config.Load("BUNAUTH_", &cfg); err != nil {
		log.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	// Safeguard: empty User can make drivers default to "postgres", which may not exist
	if cfg.DB.User == "" {
		if u := os.Getenv("BUNAUTH_DB_USER"); u != "" {
			cfg.DB.User = u
		} else {
			cfg.DB.User = "bunadmin"
		}
	}

	// 3. Connect to Database
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name)

	database, err := db.New(dsn)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	log.Info("Connected to PostgreSQL")

	// 4. Initialize RPC Server
	handler := api.NewHandler(database)

	// 5. Start HTTP Server
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Info("Listening on JSON-RPC", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
