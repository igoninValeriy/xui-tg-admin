package handlers

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	telebot "gopkg.in/telebot.v3"

	"xui-tg-admin/internal/commands"
	"xui-tg-admin/internal/config"
	"xui-tg-admin/internal/models"
	"xui-tg-admin/internal/permissions"
	"xui-tg-admin/internal/services"
)

// AdminHandler handles admin commands. Its command handlers are split across
// admin_members.go (user lifecycle) and admin_traffic.go (traffic and reporting).
type AdminHandler struct {
	BaseHandler
	commandHandlers map[string]func(context.Context, telebot.Context) error
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	xrayService *services.XrayService,
	stateService *services.UserStateService,
	qrService *services.QRService,
	config *config.Config,
	logger *logrus.Logger,
) *AdminHandler {
	baseHandler := NewBaseHandler(xrayService, stateService, qrService, config, logger)

	handler := &AdminHandler{
		BaseHandler: baseHandler,
	}

	handler.initializeCommands()
	return handler
}

// CanHandle checks if the handler can handle the given access type
func (h *AdminHandler) CanHandle(accessType permissions.AccessType) bool {
	return accessType == permissions.Admin
}

// Handle handles a message from Telegram
func (h *AdminHandler) Handle(ctx context.Context, c telebot.Context) error {
	// Get user ID
	userID := c.Sender().ID

	// Get user state
	userState, err := h.stateService.GetState(userID)
	if err != nil {
		h.logger.Errorf("Failed to get user state: %v", err)
		return err
	}

	// Handle based on state
	switch userState.State {
	case models.Default:
		return h.handleDefaultState(ctx, c)
	case models.AwaitingInputUserName:
		return h.processUserName(ctx, c)
	case models.AwaitingDuration:
		return h.processDuration(ctx, c)
	case models.AwaitSelectUserName:
		return h.processSelectUser(ctx, c)
	case models.AwaitMemberAction:
		return h.processMemberAction(ctx, c)
	case models.AwaitConfirmMemberDeletion:
		return h.processConfirmDeletion(ctx, c)
	case models.AwaitConfirmResetUsersNetworkUsage:
		return h.processConfirmResetUsersNetworkUsage(ctx, c)
	case models.AwaitUsageReportChoice:
		return h.processUsageReportChoice(ctx, c)
	default:
		h.logger.Warnf("Unknown state: %d", userState.State)
		return h.handleDefaultState(ctx, c)
	}
}

// initializeCommands initializes the command handlers
func (h *AdminHandler) initializeCommands() {
	h.commandHandlers = map[string]func(context.Context, telebot.Context) error{
		commands.Start:             h.handleStart,
		commands.AddMember:         h.handleAddMember,
		commands.EditMember:        h.handleEditMember,
		commands.DeleteMember:      h.handleDeleteMember,
		commands.OnlineMembers:     h.handleGetOnlineMembers,
		commands.DetailedUsage:     h.handleGetDetailedUsersInfo,
		commands.ResetNetworkUsage: h.handleResetUsersNetworkUsage,
		commands.ReturnToMainMenu:  h.handleStart,
		commands.Cancel:            h.handleStart,
	}
}

// getButtonCommand extracts the command from button text with emoji
func (h *AdminHandler) getButtonCommand(text string) string {
	// Check for specific button patterns
	switch text {
	case "↩️ " + commands.ReturnToMainMenu:
		return commands.ReturnToMainMenu
	case "∞ " + commands.Infinite:
		return commands.Infinite
	case "✅ " + commands.Confirm:
		return commands.Confirm
	case "❌ " + commands.Cancel:
		return commands.Cancel
	case "🔗 " + commands.ViewConfig:
		return commands.ViewConfig
	case "🔄 " + commands.ResetTraffic:
		return commands.ResetTraffic
	case "🗑️ " + commands.Delete:
		return commands.Delete
	}

	// For other buttons, try to extract command after emoji
	if len(text) > 2 && text[0] != '/' {
		if spaceIndex := strings.Index(text, " "); spaceIndex > 0 {
			return text[spaceIndex+1:]
		}
	}

	return text
}

// handleDefaultState handles the default state
func (h *AdminHandler) handleDefaultState(ctx context.Context, c telebot.Context) error {
	text := c.Text()
	command := h.getButtonCommand(text)

	// Check if we have a command handler for this command
	if handler, ok := h.commandHandlers[command]; ok {
		return handler(ctx, c)
	}

	// If not, show the main menu
	return h.handleStart(ctx, c)
}

// handleStart handles the /start command
func (h *AdminHandler) handleStart(ctx context.Context, c telebot.Context) error {
	// Clear user state
	err := h.stateService.ClearState(c.Sender().ID)
	if err != nil {
		h.logger.Errorf("Failed to clear user state: %v", err)
		return err
	}

	// Get user state
	_, err = h.stateService.GetState(c.Sender().ID)
	if err != nil {
		h.logger.Errorf("Failed to get user state: %v", err)
		return err
	}

	// Show main menu with welcome message only for /start command
	markup := h.createMainKeyboard(permissions.Admin)
	if c.Text() == commands.Start {
		return h.sendTextMessage(c, "🚀 <b>Welcome to X-UI Admin Panel!</b>\n\nYou have administrator privileges. Use the menu below to manage your VPN users, monitor connections, and configure settings.", markup)
	}

	// For return to main menu, show only the keyboard without any message
	return h.sendTextMessage(c, "🏠 <b>Main Menu</b>\n\nSelect an action:", markup)
}
