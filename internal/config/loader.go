package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("")
	v.AutomaticEnv()

	// Set default values
	v.SetDefault("log_level", "info")

	// Define environment variables
	envKeys := []string{
		"TG_TOKEN",
		"TG_ADMIN_IDS",
		"XRAY_USER",
		"XRAY_PASSWORD",
		"XRAY_API_TOKEN",
		"XRAY_API_URL",
		"XRAY_SUB_URL_PREFIX",
	}
	for _, key := range envKeys {
		if err := v.BindEnv(key); err != nil {
			return nil, fmt.Errorf("failed to bind env %s: %w", key, err)
		}
	}

	// Create config instance
	cfg := &Config{
		LogLevel: v.GetString("log_level"),
		Telegram: TelegramConfig{
			Token: v.GetString("TG_TOKEN"),
		},
	}

	// Parse admin IDs
	adminIDsStr := v.GetString("TG_ADMIN_IDS")
	if adminIDsStr != "" {
		adminIDsSlice := strings.Split(adminIDsStr, ",")
		adminIDs := make([]int64, 0, len(adminIDsSlice))
		for _, idStr := range adminIDsSlice {
			var id int64
			if _, err := fmt.Sscanf(strings.TrimSpace(idStr), "%d", &id); err == nil {
				adminIDs = append(adminIDs, id)
			}
		}
		cfg.Telegram.AdminIDs = adminIDs
	}

	// Parse server configuration
	user := strings.TrimSpace(v.GetString("XRAY_USER"))
	password := strings.TrimSpace(v.GetString("XRAY_PASSWORD"))
	apiToken := strings.TrimSpace(v.GetString("XRAY_API_TOKEN"))
	apiURL := strings.TrimSpace(v.GetString("XRAY_API_URL"))
	subURLPrefix := strings.TrimSpace(v.GetString("XRAY_SUB_URL_PREFIX"))

	// Either an API token or a username/password pair authenticates the panel.
	if apiURL == "" || (apiToken == "" && (user == "" || password == "")) {
		return nil, errors.New("missing required server configuration")
	}

	// Create server configuration
	cfg.Server = ServerConfig{
		User:         user,
		Password:     password,
		APIToken:     apiToken,
		APIURL:       apiURL,
		SubURLPrefix: subURLPrefix,
	}

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	if cfg.Telegram.Token == "" {
		return errors.New("TG_TOKEN is required")
	}

	if len(cfg.Telegram.AdminIDs) == 0 {
		return errors.New("TG_ADMIN_IDS is required")
	}

	// Validate server configuration
	if cfg.Server.APIURL == "" {
		return errors.New("XRAY_API_URL is required")
	}
	if cfg.Server.APIToken == "" {
		if cfg.Server.User == "" {
			return errors.New("XRAY_USER is required unless XRAY_API_TOKEN is set")
		}
		if cfg.Server.Password == "" {
			return errors.New("XRAY_PASSWORD is required unless XRAY_API_TOKEN is set")
		}
	}
	if cfg.Server.SubURLPrefix == "" {
		return errors.New("XRAY_SUB_URL_PREFIX is required")
	}

	return nil
}
