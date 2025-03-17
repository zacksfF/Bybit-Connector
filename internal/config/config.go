package config

import (
	"fmt"
	"log"
	"os"

	"github.com/subosito/gotenv"
)

// Configuration management
type Config struct {
	BybitWSBaseURL    string
	BybitAPIKey       string
	BybitAPISecret    string
	BybitTestnet      bool
	LogLevel          string
	ReconnectInterval int // in second
	PingInterval      int
}

// LoadConfig loads teh configuration from environment variables
func LoadConfig() (*Config, error) {
	// load .env file if it exist
	err := gotenv.Load()
	if err != nil {
		log.Println("Warning .env file not found")
	}

	conf := &Config{
		BybitWSBaseURL:    getEnv("BYBIT_WS_BASE_URL", "ws://stream.bybit.com/v5/public/spot"),
		BybitAPIKey:       getEnv("BYBIT_API_KEY", ""),
		BybitAPISecret:    getEnv("BYBIT_API_SECRET", ""),
		BybitTestnet:      getEnv("BYBIT_TESTNET", "false") == "true",
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		ReconnectInterval: getEnvAsInt("RECONNECT_INTERVAL", 5),
		PingInterval:      getEnvAsInt("PIN_INTERVAL", 20),
	}

	//adjust URL if using testnet
	if conf.BybitTestnet {
		conf.BybitWSBaseURL = "ws://stream.bybit.com/v5/public/spot"
	}
	return conf, nil
}

// getEnv gets an environmebt variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaulValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaulValue
	}

	value := 0
	_, err := fmt.Scanf(valueStr, "%d", &value)
	if err != nil {
		log.Printf("warning: invalid value for %s, using default %d", key, defaulValue)
		return defaulValue
	}
	return value
}
