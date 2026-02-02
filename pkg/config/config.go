package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load loads configuration from .env file and environment variables
// prefix: Environment variable prefix (e.g. "BUNBASE_")
// target: Pointer to the config struct to load into
func Load(prefix string, target interface{}) error {
	v := viper.New()

	// 1. Load from .env file (if exists)
	v.SetConfigFile(".env")
	if err := v.ReadInConfig(); err != nil {
		// Ignore error if file doesn't exist, it's optional
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// If it's another error (e.g. parsing), we might want to log it but carrying on is standard if optional.
			// formatting error might catch later during Unmarshal if critical.
		}
	}

	// 2. Load from environment variables
	// Viper's AutomaticEnv doesn't work well with Unmarshal if keys aren't known (e.g. no config file).
	// We mimic koanf's env.Provider: iterate env vars and populate viper.

	prefixUpper := strings.ToUpper(prefix)
	for _, envStr := range os.Environ() {
		pair := strings.SplitN(envStr, "=", 2)
		key, value := pair[0], pair[1]

		if strings.HasPrefix(key, prefixUpper) {
			// BUNAUTH_DB_HOST -> db.host
			propKey := strings.TrimPrefix(key, prefixUpper)
			propKey = strings.ToLower(strings.ReplaceAll(propKey, "_", "."))
			// Remove leading dot if any (e.g. if prefix didn't include underscore but env did)
			propKey = strings.TrimPrefix(propKey, ".")

			v.Set(propKey, value)
		}
	}

	// 3. Unmarshal into struct
	if err := v.Unmarshal(target); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}
