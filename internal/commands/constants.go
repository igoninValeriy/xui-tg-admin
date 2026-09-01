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
	NodeStatus        = "Node Status"
	Subscriptions     = "Subscriptions"
	DetailedUsage     = "Detailed Usage"
	ResetNetworkUsage = "Reset Network Usage"

	// Detailed Usage sub-menu
	UsageTable = "Usage Table"
	UsagePhoto = "Usage Photo"

	// Member action commands
	ViewConfig     = "View Config"
	SubscriptionQR = "Subscription QR"
	ResetTraffic   = "Reset Traffic"
	Delete         = "Delete"

	// Confirmation commands
	Confirm = "Confirm"

	// Duration options
	Infinite = "Infinite"
)

// Callback data uniques for inline buttons.
const (
	// CallbackNodeStatusRefresh redraws the node status message in place.
	CallbackNodeStatusRefresh = "node_status_refresh"
)
