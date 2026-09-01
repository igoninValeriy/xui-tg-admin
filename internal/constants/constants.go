package constants

import "time"

const (
	// User validation constants
	MaxUsernameLength = 64

	// User naming constants
	UsernameSeparator = "-"

	// Traffic constants
	BytesInGB = 1024 * 1024 * 1024

	// Duration constants
	MillisecondsInDay = 24 * 60 * 60 * 1000
	MinDurationDays   = 1
	MaxDurationDays   = 3650 // 10 years

	// Network constants
	DefaultTimeout          = 30
	DefaultRetryCount       = 3
	DefaultRetryWaitTime    = 5
	DefaultRetryMaxWaitTime = 20

	// Cache constants
	CacheExpiration      = 30 // minutes
	CacheCleanupInterval = 10 // minutes

	// Formatting constants
	TimestampFormat = "2006-01-02 15:04:05"
	DateFormat      = "2006-01-02"
)

// Node probes. The bot runs on the same host as the VPN tooling, so these
// screens shell out to the scripts and files that already own the answer
// instead of carrying a second copy of their logic. Every path lives here so a
// server-side rename is a one-line change.
const (
	// NodeDestFile holds the REALITY donor currently in use.
	NodeDestFile = "/etc/vpn-dest"
	// NodeWatchdogLog is the watchdog's append-only log; its last line is the
	// most recent check.
	NodeWatchdogLog = "/var/log/vpn-watchdog.log"
	// NodeSubStatsScript prints one line per device: request count, last seen
	// and the countries it was requested from.
	NodeSubStatsScript = "/usr/local/bin/vpn-sub-stats"

	// NodeChannelPort is the TCP port the VPN channel listens on.
	NodeChannelPort = "443"
)

// NodeServices are the systemd units reported on the node status screen.
var NodeServices = []string{"x-ui", "vpn-watchdog.timer", "fail2ban"}

const (
	// NodeCommandTimeout caps a single local probe (ss, systemctl, free, df).
	NodeCommandTimeout = 10 * time.Second
	// NodeSubStatsTimeout caps the subscription statistics script, which talks
	// to a remote KV store once per device and is therefore much slower.
	NodeSubStatsTimeout = 120 * time.Second

	// MaxTelegramMessageLength leaves room for the HTML wrapper inside
	// Telegram's 4096-character limit.
	MaxTelegramMessageLength = 3800
)
