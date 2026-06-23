package handlers

import (
	"context"
	"fmt"
	"strings"

	telebot "gopkg.in/telebot.v3"

	"xui-tg-admin/internal/commands"
	"xui-tg-admin/internal/helpers"
	"xui-tg-admin/internal/models"
	"xui-tg-admin/internal/permissions"
)

// handleGetOnlineMembers handles the Online Members command
func (h *AdminHandler) handleGetOnlineMembers(ctx context.Context, c telebot.Context) error {

	// Get online users
	onlineUsers, err := h.xrayService.GetOnlineUsers(ctx)
	if err != nil {
		h.logger.Errorf("Failed to get online users: %v", err)
		return h.sendTextMessage(c, "❌ <b>Connection Error</b>\n\nCouldn't retrieve online users. Please check your server connection and try again.", h.createMainKeyboard(permissions.Admin))
	}

	// Format message
	var message string
	if len(onlineUsers) == 0 {
		message = "💤 <b>No Active Connections</b>\n\nNo users are currently connected to the VPN server."
	} else {
		message = fmt.Sprintf("🟢 <b>Active Connections (%d)</b>\n\n", len(onlineUsers))
		for _, user := range onlineUsers {
			message += fmt.Sprintf("👤 %s\n", user)
		}
	}

	return h.sendTextMessage(c, message, h.createMainKeyboard(permissions.Admin))
}

// handleResetUsersNetworkUsage handles the Reset Network Usage command
func (h *AdminHandler) handleResetUsersNetworkUsage(ctx context.Context, c telebot.Context) error {
	// Set state to awaiting confirmation for reset
	err := h.stateService.WithConversationState(c.Sender().ID, models.AwaitConfirmResetUsersNetworkUsage)
	if err != nil {
		h.logger.Errorf("Failed to set state: %v", err)
		return err
	}

	// Show confirm keyboard
	markup := h.createConfirmKeyboard()
	return h.sendTextMessage(c, "⚠️ <b>Reset All Network Usage</b>\n\nThis will reset traffic statistics for <b>ALL users</b> in the system.\n\n<b>⚠️ This action cannot be undone!</b>\n\nAre you sure you want to proceed?", markup)
}

// handleResetTraffic handles the Reset Traffic action
func (h *AdminHandler) handleResetTraffic(ctx context.Context, c telebot.Context, username string) error {
	h.logger.Infof("Starting reset traffic for user: %s", username)

	// Send loading message
	loadingMsg, _ := h.sendTextMessageWithReturn(c, fmt.Sprintf("⏳ <b>Resetting Traffic...</b>\n\nResetting traffic statistics for user '%s'. Please wait...", username), nil)

	// Get all inbounds
	inbounds, err := h.xrayService.GetInbounds(ctx)
	if err != nil {
		h.logger.Errorf("Failed to get inbounds: %v", err)
		return h.sendTextMessage(c, "❌ <b>Connection Error</b>\n\nCouldn't retrieve server data. Please check your connection and try again.", h.createUserActionKeyboard())
	}

	// Find all clients with the base username and reset their traffic
	var resetErrors []string
	successfullyReset := 0

	for _, inbound := range inbounds {
		for _, clientStat := range inbound.ClientStats {
			// Check if client email matches the base username using helper function
			if helpers.IsEmailMatchingBaseUsername(clientStat.Email, username) {
				h.logger.Infof("Found matching client: %s in inbound %d", clientStat.Email, inbound.ID)

				err := h.xrayService.ResetUserTraffic(ctx, inbound.ID, clientStat.Email)
				if err != nil {
					h.logger.Errorf("Failed to reset traffic for %s in inbound %d: %v", clientStat.Email, inbound.ID, err)
					resetErrors = append(resetErrors, fmt.Sprintf("Failed to reset %s in inbound %d: %v", clientStat.Email, inbound.ID, err))
				} else {
					h.logger.Infof("Successfully reset traffic for %s in inbound %d", clientStat.Email, inbound.ID)
					successfullyReset++
				}
			}
		}
	}

	// Send result message
	var message string
	if successfullyReset > 0 {
		message = fmt.Sprintf("✅ <b>Traffic Reset Complete</b>\n\n🔄 Successfully reset traffic for user <b>%s</b> (%d configurations)", username, successfullyReset)
		if len(resetErrors) > 0 {
			message += fmt.Sprintf("\n\n⚠️ <b>Some errors occurred:</b>\n%s", strings.Join(resetErrors, "\n"))
		}
	} else {
		message = fmt.Sprintf("❌ <b>Reset Failed</b>\n\nNo active configurations found for user '%s'.", username)
		if len(resetErrors) > 0 {
			message += fmt.Sprintf("\n\n<b>Errors:</b>\n%s", strings.Join(resetErrors, "\n"))
		}
	}

	// Delete loading message
	if loadingMsg != nil {
		c.Bot().Delete(loadingMsg)
	}

	return h.sendTextMessage(c, message, h.createUserActionKeyboard())
}

// handleGetDetailedUsersInfo handles the Detailed Usage command
func (h *AdminHandler) handleGetDetailedUsersInfo(ctx context.Context, c telebot.Context) error {

	// Get inbounds
	inbounds, err := h.xrayService.GetInbounds(ctx)
	if err != nil {
		h.logger.Errorf("Failed to get inbounds: %v", err)
		return h.sendTextMessage(c, "❌ <b>Connection Error</b>\n\nCouldn't retrieve detailed usage data. Please check your server connection and try again.", h.createMainKeyboard(permissions.Admin))
	}

	// Get online users for status indication
	onlineUsers, err := h.xrayService.GetOnlineUsers(ctx)
	if err != nil {
		h.logger.Errorf("Failed to get online users: %v", err)
		// Continue with empty online users list if this fails
		onlineUsers = []string{}
	}

	// Format compact traffic report
	message := helpers.FormatCompactTrafficReport(inbounds, onlineUsers)

	return h.sendTextMessage(c, message, h.createMainKeyboard(permissions.Admin))
}

// processConfirmResetUsersNetworkUsage processes the confirmation for resetting network usage
func (h *AdminHandler) processConfirmResetUsersNetworkUsage(ctx context.Context, c telebot.Context) error {
	// Get confirmation from message
	confirmation := c.Text()

	// Check for return to main menu
	if h.getButtonCommand(confirmation) == commands.ReturnToMainMenu {
		return h.handleStart(ctx, c)
	}

	// Check if user confirmed
	if h.getButtonCommand(confirmation) != commands.Confirm {
		return h.sendTextMessage(c, "❌ <b>Invalid Selection</b>\n\nPlease click Confirm to proceed with reset or use the Return button to cancel.", h.createConfirmKeyboard())
	}

	h.logger.Infof("Starting reset network usage for all users")

	// Send loading message
	loadingMsg, _ := h.sendTextMessageWithReturn(c, "⏳ <b>Resetting All Traffic...</b>\n\nThis may take a few moments. Resetting traffic statistics for all users across all servers...", nil)

	// Get all inbounds
	inbounds, err := h.xrayService.GetInbounds(ctx)
	if err != nil {
		h.logger.Errorf("Failed to get inbounds: %v", err)
		return h.sendTextMessage(c, "❌ <b>Connection Error</b>\n\nCouldn't retrieve server data for reset operation. Please check your connection and try again.", h.createMainKeyboard(permissions.Admin))
	}

	// Collect all user emails from all inbounds
	var userEmails []struct {
		inboundID int
		email     string
	}

	for _, inbound := range inbounds {
		for _, clientStat := range inbound.ClientStats {
			userEmails = append(userEmails, struct {
				inboundID int
				email     string
			}{
				inboundID: inbound.ID,
				email:     clientStat.Email,
			})
		}
	}

	if len(userEmails) == 0 {
		return h.sendTextMessage(c, "📭 <b>No Users Found</b>\n\nThere are no users in the system to reset traffic for.", h.createMainKeyboard(permissions.Admin))
	}

	h.logger.Infof("Found %d users to reset traffic", len(userEmails))

	// Reset traffic for all users
	var resetErrors []string
	successfullyReset := 0

	for _, user := range userEmails {
		err := h.xrayService.ResetUserTraffic(ctx, user.inboundID, user.email)
		if err != nil {
			h.logger.Errorf("Failed to reset traffic for %s in inbound %d: %v", user.email, user.inboundID, err)
			resetErrors = append(resetErrors, fmt.Sprintf("Failed to reset %s in inbound %d: %v", user.email, user.inboundID, err))
		} else {
			h.logger.Infof("Successfully reset traffic for %s in inbound %d", user.email, user.inboundID)
			successfullyReset++
		}
	}

	// Send result message
	var message string
	if successfullyReset > 0 {
		message = fmt.Sprintf("✅ <b>Mass Traffic Reset Complete</b>\n\n🔄 Successfully reset traffic for <b>%d users</b>\n\n<i>All user traffic counters have been set to zero</i>", successfullyReset)
		if len(resetErrors) > 0 {
			message += fmt.Sprintf("\n\n⚠️ <b>Some errors occurred:</b>\n%s", strings.Join(resetErrors, "\n"))
		}
	} else {
		message = fmt.Sprintf("❌ <b>Mass Reset Failed</b>\n\nCouldn't reset traffic for any users.\n\n<b>Errors:</b>\n%s", strings.Join(resetErrors, "\n"))
	}

	// Delete loading message
	if loadingMsg != nil {
		c.Bot().Delete(loadingMsg)
	}

	// Clear user state and return to main menu
	err = h.stateService.ClearState(c.Sender().ID)
	if err != nil {
		h.logger.Errorf("Failed to clear user state: %v", err)
	}

	return h.sendTextMessage(c, message, h.createMainKeyboard(permissions.Admin))
}
