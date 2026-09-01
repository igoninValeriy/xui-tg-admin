package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	telebot "gopkg.in/telebot.v3"

	"xui-tg-admin/internal/commands"
	"xui-tg-admin/internal/helpers"
	"xui-tg-admin/internal/models"
	"xui-tg-admin/internal/permissions"
	"xui-tg-admin/internal/render"
)

// handleGetOnlineMembers lists who is connected right now and, for everyone
// else, when they were last seen.
//
// Panel 3.7.0 answers both questions: POST /panel/api/clients/onlines returns
// the emails seen within the heartbeat window, and POST
// /panel/api/clients/lastOnline maps every email to its last-seen timestamp.
// The legacy /xui/API/inbounds/onlines endpoint this bot used is gone.
func (h *AdminHandler) handleGetOnlineMembers(ctx context.Context, c telebot.Context) error {
	mainKeyboard := h.createMainKeyboard(permissions.Admin)

	onlineUsers, err := h.xrayService.GetOnlineUsers(ctx)
	if err != nil {
		h.logger.Errorf("Failed to get online users: %v", err)
		return h.sendTextMessage(c, "❌ <b>Connection Error</b>\n\nCouldn't retrieve online users. Please check your server connection and try again.", mainKeyboard)
	}

	// Last-seen data is a nice-to-have: without it the list still answers the
	// actual question, so a failure here only drops the extra column.
	members, err := h.xrayService.GetAllMembersWithInfo(ctx, models.SortByLastOnline)
	if err != nil {
		h.logger.Errorf("Failed to get member info: %v", err)
		members = nil
	}

	onlineSet := make(map[string]bool, len(onlineUsers))
	for _, email := range onlineUsers {
		onlineSet[helpers.ExtractBaseUsername(email)] = true
	}

	var sb strings.Builder
	if len(onlineUsers) == 0 {
		sb.WriteString("💤 <b>No Active Connections</b>\n\nNo users are currently connected to the VPN server.\n")
	} else {
		sb.WriteString(fmt.Sprintf("🟢 <b>Active Connections (%d)</b>\n\n", len(onlineUsers)))
		for _, email := range onlineUsers {
			sb.WriteString(fmt.Sprintf("👤 %s\n", email))
		}
	}

	// Everyone who is not connected right now, most recently seen first.
	var offline []models.MemberInfo
	for _, member := range members {
		if !onlineSet[member.BaseUsername] {
			offline = append(offline, member)
		}
	}
	if len(offline) > 0 {
		sb.WriteString(fmt.Sprintf("\n⚪️ <b>Last seen (%d)</b>\n\n", len(offline)))
		for _, member := range offline {
			sb.WriteString(fmt.Sprintf("• %s — %s\n", member.BaseUsername, member.LastSeenStatus()))
		}
	}

	return h.sendTextMessage(c, sb.String(), mainKeyboard)
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

// handleResetTraffic resets one user's traffic. Since 3.7.0 traffic belongs to
// the client, not to an inbound, so a single call covers every inbound the
// client is attached to instead of one call per inbound.
func (h *AdminHandler) handleResetTraffic(ctx context.Context, c telebot.Context, username string) error {
	h.logger.Infof("Starting reset traffic for user: %s", username)

	loadingMsg, _ := h.sendTextMessageWithReturn(c, fmt.Sprintf("⏳ <b>Resetting Traffic...</b>\n\nResetting traffic statistics for user '%s'. Please wait...", username), nil)
	defer func() {
		if loadingMsg != nil {
			c.Bot().Delete(loadingMsg)
		}
	}()

	clients, err := h.xrayService.FindMemberClients(ctx, username)
	if err != nil {
		h.logger.Errorf("Failed to look up client %s: %v", username, err)
		return h.sendTextMessage(c, "❌ <b>Connection Error</b>\n\nCouldn't retrieve server data. Please check your connection and try again.", h.createUserActionKeyboard())
	}
	if len(clients) == 0 {
		return h.sendTextMessage(c, fmt.Sprintf("❌ <b>Reset Failed</b>\n\nNo configurations found for user '%s'.", username), h.createUserActionKeyboard())
	}

	var resetErrors []string
	successfullyReset := 0
	for _, record := range clients {
		if err := h.xrayService.ResetClientTraffic(ctx, record.Email); err != nil {
			h.logger.Errorf("Failed to reset traffic for %s: %v", record.Email, err)
			resetErrors = append(resetErrors, fmt.Sprintf("%s: %v", record.Email, err))
			continue
		}
		h.logger.Infof("Successfully reset traffic for %s", record.Email)
		successfullyReset++
	}

	var message string
	if successfullyReset > 0 {
		message = fmt.Sprintf("✅ <b>Traffic Reset Complete</b>\n\n🔄 Successfully reset traffic for user <b>%s</b> (%d configurations)", username, successfullyReset)
		if len(resetErrors) > 0 {
			message += fmt.Sprintf("\n\n⚠️ <b>Some errors occurred:</b>\n%s", strings.Join(resetErrors, "\n"))
		}
	} else {
		message = fmt.Sprintf("❌ <b>Reset Failed</b>\n\nCouldn't reset traffic for user '%s'.\n\n<b>Errors:</b>\n%s", username, strings.Join(resetErrors, "\n"))
	}

	return h.sendTextMessage(c, message, h.createUserActionKeyboard())
}

// handleGetDetailedUsersInfo shows the Detailed Usage sub-menu (Table / Photo / Back).
func (h *AdminHandler) handleGetDetailedUsersInfo(ctx context.Context, c telebot.Context) error {
	if err := h.stateService.WithConversationState(c.Sender().ID, models.AwaitUsageReportChoice); err != nil {
		h.logger.Errorf("Failed to set state: %v", err)
		return err
	}
	return h.sendTextMessage(c, "📈 <b>Detailed Usage</b>\n\nChoose how to view the traffic report:", h.createUsageReportKeyboard())
}

// processUsageReportChoice handles the Detailed Usage sub-menu selection. Every
// branch returns the user to the main menu when done.
func (h *AdminHandler) processUsageReportChoice(ctx context.Context, c telebot.Context) error {
	switch h.getButtonCommand(c.Text()) {
	case commands.UsageTable:
		return h.sendUsageReport(ctx, c, false)
	case commands.UsagePhoto:
		return h.sendUsageReport(ctx, c, true)
	case commands.ReturnToMainMenu:
		return h.handleStart(ctx, c)
	default:
		return h.sendTextMessage(c, "❓ Please choose <b>Table</b>, <b>Photo</b> or go back.", h.createUsageReportKeyboard())
	}
}

// sendUsageReport fetches traffic data and sends it as a rendered image (asPhoto)
// or as a text table, then returns the user to the main menu.
func (h *AdminHandler) sendUsageReport(ctx context.Context, c telebot.Context, asPhoto bool) error {
	mainKeyboard := h.createMainKeyboard(permissions.Admin)

	if err := h.stateService.ClearState(c.Sender().ID); err != nil {
		h.logger.Errorf("Failed to clear user state: %v", err)
	}

	inbounds, err := h.xrayService.GetInbounds(ctx)
	if err != nil {
		h.logger.Errorf("Failed to get inbounds: %v", err)
		return h.sendTextMessage(c, "❌ <b>Connection Error</b>\n\nCouldn't retrieve usage data. Please check your server connection and try again.", mainKeyboard)
	}

	onlineUsers, err := h.xrayService.GetOnlineUsers(ctx)
	if err != nil {
		h.logger.Errorf("Failed to get online users: %v", err)
		onlineUsers = []string{}
	}

	report := helpers.AggregateTraffic(inbounds, onlineUsers)
	now := time.Now()

	// On a render failure we fall back to the text table, so the admin is never
	// left empty-handed.
	if asPhoto {
		img, rerr := render.TrafficReport(report, now)
		if rerr == nil {
			return h.sendPhotoBytes(c, img, mainKeyboard)
		}
		h.logger.Errorf("Failed to render traffic report image, falling back to text: %v", rerr)
	}

	return h.sendTextMessage(c, helpers.FormatTrafficText(report, now), mainKeyboard)
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
	loadingMsg, _ := h.sendTextMessageWithReturn(c, "⏳ <b>Resetting All Traffic...</b>\n\nThis may take a few moments. Resetting traffic statistics for all users...", nil)

	// The panel resets every client in one call, so there is no partial state
	// to report and no per-client loop to get half-way through.
	resetErr := h.xrayService.ResetAllTraffics(ctx)

	// Delete loading message
	if loadingMsg != nil {
		c.Bot().Delete(loadingMsg)
	}

	var message string
	if resetErr != nil {
		h.logger.Errorf("Failed to reset all traffic: %v", resetErr)
		message = fmt.Sprintf("❌ <b>Mass Reset Failed</b>\n\nCouldn't reset traffic counters.\n\n<b>Error:</b> %v", resetErr)
	} else {
		message = "✅ <b>Mass Traffic Reset Complete</b>\n\n🔄 Traffic counters were reset for every user\n\n<i>Quotas and expiry dates are untouched</i>"
	}

	// Clear user state and return to main menu
	if err := h.stateService.ClearState(c.Sender().ID); err != nil {
		h.logger.Errorf("Failed to clear user state: %v", err)
	}

	return h.sendTextMessage(c, message, h.createMainKeyboard(permissions.Admin))
}
