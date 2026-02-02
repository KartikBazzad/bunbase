package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// Config holds BunKMS configuration (env and optional file).
type Config struct {
	Addr       string
	MasterKey  string
	DataPath   string
	AuditLog   string
	JWTSecret  string
	BufferPool int
	Shards     int
}

// Load reads configuration from environment variables.
// BUNKMS_ADDR, BUNKMS_MASTER_KEY, BUNKMS_DATA_PATH, BUNKMS_AUDIT_LOG, BUNKMS_JWT_SECRET,
// BUNKMS_BUFFER_POOL_SIZE, BUNKMS_SHARDS.
func Load() *Config {
	c := &Config{
		Addr:       getEnv("BUNKMS_ADDR", ":8080"),
		MasterKey:  os.Getenv("BUNKMS_MASTER_KEY"),
		DataPath:   os.Getenv("BUNKMS_DATA_PATH"),
		AuditLog:   os.Getenv("BUNKMS_AUDIT_LOG"),
		JWTSecret:  os.Getenv("BUNKMS_JWT_SECRET"),
		BufferPool: getEnvInt("BUNKMS_BUFFER_POOL_SIZE", 10000),
		Shards:     getEnvInt("BUNKMS_SHARDS", 256),
	}
	return c
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return strings.TrimSpace(v)
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return n
		}
	}
	return defaultVal
}

// Validate returns an error if required fields are missing.
func (c *Config) Validate() error {
	if c.MasterKey == "" {
		return ErrMasterKeyRequired
	}
	return nil
}

// ErrMasterKeyRequired is returned when BUNKMS_MASTER_KEY is not set.
var ErrMasterKeyRequired = errors.New("BUNKMS_MASTER_KEY is required")
