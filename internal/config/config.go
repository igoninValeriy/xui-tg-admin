package config

// Config represents the application configuration
type Config struct {
	Telegram TelegramConfig `mapstructure:"telegram"`
	Server   ServerConfig   `mapstructure:"server"`
	LogLevel string         `mapstructure:"log_level"`
}

// TelegramConfig holds the Telegram bot configuration
type TelegramConfig struct {
	Token    string  `mapstructure:"token"`
	AdminIDs []int64 `mapstructure:"admin_ids"`
}

// ServerConfig holds the configuration for one 3x-ui panel.
type ServerConfig struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	// APIToken is an optional panel API token (Settings -> Security -> API
	// Token). When set it replaces the username/password session and skips the
	// CSRF handshake, because the panel short-circuits CSRF for Bearer callers.
	APIToken string `mapstructure:"api_token"`
	APIURL   string `mapstructure:"api_url"`
	// SubURLPrefix is the base URL that serves subscriptions. It is not
	// necessarily the panel: here subscriptions are served by a Cloudflare
	// Worker, and the full URL is this prefix joined with the client's SubID.
	SubURLPrefix string `mapstructure:"sub_url_prefix"`
}
