package commands

// TelegramCommands contains all commands for the Telegram bot
const (
	// Main commands
	Start  = "/start"
	Cancel = "Cancel"

	// Navigation commands
	ReturnToMainMenu = "Return to Main Menu"

	// Administrator commands
	AddMember         = "Add Member"
	EditMember        = "Edit Member"
	DeleteMember      = "Delete Member"
	OnlineMembers     = "Online Members"
	DetailedUsage     = "Detailed Usage"
	ResetNetworkUsage = "Reset Network Usage"

	// Member action commands
	ViewConfig   = "View Config"
	ResetTraffic = "Reset Traffic"
	Delete       = "Delete"

	// Confirmation commands
	Confirm = "Confirm"

	// Duration options
	Infinite = "Infinite"
)
