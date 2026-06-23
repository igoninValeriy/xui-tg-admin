package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	telebot "gopkg.in/telebot.v3"

	"xui-tg-admin/internal/commands"
	"xui-tg-admin/internal/helpers"
	"xui-tg-admin/internal/models"
	"xui-tg-admin/internal/validation"
)

// handleAddMember handles the Add Member command
func (h *AdminHandler) handleAddMember(ctx context.Context, c telebot.Context) error {

	// Set state to awaiting username
	err := h.stateService.WithConversationState(c.Sender().ID, models.AwaitingInputUserName)
	if err != nil {
		h.logger.Errorf("Failed to set state: %v", err)
		return err
	}

	// Show return keyboard
	markup := h.createReturnKeyboard()
	return h.sendTextMessage(c, "👤 <b>Add New User</b>\n\n📝 Please enter a username for the new user:\n\n<i>• Letters, numbers and - . _ ~\n• Up to 64 characters\n• Example: john_doe, user-123</i>", markup)
}

// handleEditMember handles the Edit Member command
func (h *AdminHandler) handleEditMember(ctx context.Context, c telebot.Context) error {
	// Проверяем доступность сервиса
	_, err := h.stateService.GetState(c.Sender().ID)
	if err != nil {
		h.logger.Errorf("Failed to get user state: %v", err)
		return err
	}

	// Показываем список пользователей с сортировкой по дате добавления
	return h.showMembersWithSort(ctx, c, models.SortByCreationOrder, "edit")
}

// handleDeleteMember handles the Delete Member command
func (h *AdminHandler) handleDeleteMember(ctx context.Context, c telebot.Context) error {
	// Проверяем доступность сервиса
	_, err := h.stateService.GetState(c.Sender().ID)
	if err != nil {
		h.logger.Errorf("Failed to get user state: %v", err)
		return err
	}

	// Показываем список пользователей с сортировкой по дате добавления
	return h.showMembersWithSort(ctx, c, models.SortByCreationOrder, "delete")
}

// processUserName processes the username input
func (h *AdminHandler) processUserName(ctx context.Context, c telebot.Context) error {
	// Get username from message
	username := strings.TrimSpace(c.Text())

	// Check for return to main menu
	if h.getButtonCommand(username) == commands.ReturnToMainMenu {
		return h.handleStart(ctx, c)
	}

	// Validate username format
	if err := validation.ValidateUsername(username); err != nil {
		return h.sendTextMessage(c, fmt.Sprintf("❌ <b>Invalid Username</b>\n\n%s\n\n💡 Allowed: letters, numbers and <code>- . _ ~</code> (up to 64 characters)\n\nPlease try again:", err.Error()), h.createReturnKeyboard())
	}

	// Store username in state
	err := h.stateService.WithPayload(c.Sender().ID, username)
	if err != nil {
		h.logger.Errorf("Failed to set payload: %v", err)
		return err
	}

	// Set state to awaiting duration
	err = h.stateService.WithConversationState(c.Sender().ID, models.AwaitingDuration)
	if err != nil {
		h.logger.Errorf("Failed to set state: %v", err)
		return err
	}

	// Create keyboard with Infinite option
	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}
	markup.Reply(
		telebot.Row{
			telebot.Btn{Text: "∞ " + commands.Infinite},
		},
		telebot.Row{
			telebot.Btn{Text: "↩️ " + commands.ReturnToMainMenu},
		},
	)

	return h.sendTextMessage(c, fmt.Sprintf("⏰ <b>Set Duration for %s</b>\n\n📅 Enter subscription duration in days:\n\n<i>• Example: 30 (for 30 days)\n• Maximum: 3650 days\n• Or choose Infinite for unlimited time</i>", username), markup)
}

// processDuration processes the duration input
func (h *AdminHandler) processDuration(ctx context.Context, c telebot.Context) error {
	// Get duration from message
	durationStr := c.Text()

	// Check for return to main menu
	if h.getButtonCommand(durationStr) == commands.ReturnToMainMenu {
		return h.handleStart(ctx, c)
	}

	// Extract command from button text
	durationStr = h.getButtonCommand(durationStr)

	// Get user state
	userState, err := h.stateService.GetState(c.Sender().ID)
	if err != nil {
		h.logger.Errorf("Failed to get user state: %v", err)
		return err
	}

	// Get username from state
	if userState.Payload == nil {
		return h.sendTextMessage(c, "❌ <b>Session Error</b>\n\nUsername data was lost. Please start over.", h.createReturnKeyboard())
	}

	baseUsername := *userState.Payload

	// Get enabled inbounds
	enabledInbounds, err := h.getEnabledInbounds(ctx)
	if err != nil {
		h.logger.Errorf("Failed to get enabled inbounds: %v", err)
		return h.sendTextMessage(c, "❌ <b>Server Configuration Error</b>\n\nNo enabled inbound connections found. Please check your server configuration or contact the administrator.", h.createReturnKeyboard())
	}

	// Calculate expiry time
	expiryTime, err := calculateExpiryTime(durationStr)
	if err != nil {
		return h.sendTextMessage(c, fmt.Sprintf("❌ <b>Invalid Duration</b>\n\n%s\n\n💡 <b>Valid formats:</b>\n• Number: 30 (for 30 days)\n• Range: 1-3650 days\n• Or use the Infinite button\n\nPlease try again:", err.Error()), h.createReturnKeyboard())
	}

	// Create client creation parameters
	params := ClientCreationParams{
		BaseUsername:    baseUsername,
		DurationStr:     durationStr,
		ExpiryTime:      expiryTime,
		CommonSubId:     models.GenerateSubID(),
		BaseFingerprint: fmt.Sprintf("%x", time.Now().UnixNano()),
		SenderID:        c.Sender().ID,
	}

	// Send loading message
	loadingMsg, _ := h.sendTextMessageWithReturn(c, "⏳ <b>Creating User...</b>\n\nPlease wait while we set up the new user configuration across all servers.", nil)

	// Create clients for all enabled inbounds
	createdEmails, addErrors, addedToAny := h.createClientsForAllInbounds(ctx, params, enabledInbounds)

	// Delete loading message
	if loadingMsg != nil {
		c.Bot().Delete(loadingMsg)
	}

	if !addedToAny {
		return h.sendTextMessage(c, fmt.Sprintf("❌ <b>User Creation Failed</b>\n\nCouldn't create user '%s' in any server configuration.\n\n<b>Errors:</b>\n%s\n\nPlease check server configuration or try again later.", baseUsername, strings.Join(addErrors, "\n")), h.createReturnKeyboard())
	}

	// Send subscription information and QR code
	return h.sendSubscriptionInfo(c, params, createdEmails, addErrors)
}

// processSelectUser processes the user selection
func (h *AdminHandler) processSelectUser(ctx context.Context, c telebot.Context) error {
	// Get username from message
	username := c.Text()

	// Check for return to main menu
	if h.getButtonCommand(username) == commands.ReturnToMainMenu {
		return h.handleStart(ctx, c)
	}

	// Store username in state
	err := h.stateService.WithPayload(c.Sender().ID, username)
	if err != nil {
		h.logger.Errorf("Failed to set payload: %v", err)
		return err
	}

	// Set state to awaiting member action
	err = h.stateService.WithConversationState(c.Sender().ID, models.AwaitMemberAction)
	if err != nil {
		h.logger.Errorf("Failed to set state: %v", err)
		return err
	}

	// Create action keyboard
	markup := h.createUserActionKeyboard()

	return h.sendTextMessage(c, fmt.Sprintf("👤 <b>Managing User: %s</b>\n\n🎛️ Choose an action:", username), markup)
}

// processMemberAction processes the member action selection
func (h *AdminHandler) processMemberAction(ctx context.Context, c telebot.Context) error {
	// Get action from message
	action := c.Text()

	// Check for return to main menu first
	if h.getButtonCommand(action) == commands.ReturnToMainMenu {
		return h.handleStart(ctx, c)
	}

	// Проверяем доступность сервиса
	userState, err := h.stateService.GetState(c.Sender().ID)
	if err != nil {
		h.logger.Errorf("Failed to get user state: %v", err)
		return err
	}

	// Get username from state
	if userState.Payload == nil {
		return h.sendTextMessage(c, "❌ <b>Session Error</b>\n\nUser data was lost. Please start over.", h.createReturnKeyboard())
	}

	username := *userState.Payload

	// Extract command from button text
	command := h.getButtonCommand(action)

	// Handle action
	switch command {
	case commands.ViewConfig:
		return h.handleViewConfig(ctx, c, username)
	case commands.ResetTraffic:
		return h.handleResetTraffic(ctx, c, username)
	case commands.Delete:
		return h.handleConfirmDelete(c, username)
	default:
		return h.sendTextMessage(c, "❌ <b>Invalid Action</b>\n\nPlease select one of the available options from the menu.", h.createUserActionKeyboard())
	}
}

// createUserActionKeyboard creates a keyboard for user actions
func (h *AdminHandler) createUserActionKeyboard() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}

	markup.Reply(
		telebot.Row{
			telebot.Btn{Text: "🔗 " + commands.ViewConfig},
		},
		telebot.Row{
			telebot.Btn{Text: "🔄 " + commands.ResetTraffic},
			telebot.Btn{Text: "🗑️ " + commands.Delete},
		},
		telebot.Row{
			telebot.Btn{Text: "↩️ " + commands.ReturnToMainMenu},
		},
	)

	return markup
}

// handleViewConfig handles the View Config action
func (h *AdminHandler) handleViewConfig(ctx context.Context, c telebot.Context, username string) error {
	h.logger.Infof("Starting view config for user: %s", username)

	// Get all inbounds
	inbounds, err := h.xrayService.GetInbounds(ctx)
	if err != nil {
		h.logger.Errorf("Failed to get inbounds: %v", err)
		return h.sendTextMessage(c, fmt.Sprintf("Failed to get inbounds: %v", err), h.createUserActionKeyboard())
	}

	// Find first client with the base username to get SubID
	var foundClientSubID string

	for _, inbound := range inbounds {
		// Parse inbound settings to get client details
		var settings models.InboundSettings
		if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
			h.logger.Errorf("Failed to parse settings for inbound %d: %v", inbound.ID, err)
			continue
		}

		// Find client in settings
		for _, client := range settings.Clients {
			// Check if client email matches the base username using helper function
			if helpers.IsEmailMatchingBaseUsername(client.Email, username) {
				h.logger.Infof("Found matching client: %s in inbound %d", client.Email, inbound.ID)
				foundClientSubID = client.SubID
				break
			}
		}
		if foundClientSubID != "" {
			break
		}
	}

	if foundClientSubID == "" {
		return h.sendTextMessage(c, fmt.Sprintf("❌ <b>User Not Found</b>\n\nNo configuration found for user '%s'. The user may have been deleted or never existed.", username), h.createUserActionKeyboard())
	}

	// Get subscription URL using SubID (same format as when adding user)
	subURL := fmt.Sprintf("%s%s?name=%s", h.config.Server.SubURLPrefix, foundClientSubID, foundClientSubID)

	// Send subscription URL with user action keyboard (stays in same state)
	err = h.sendTextMessage(c, fmt.Sprintf("🔗 <b>Configuration for %s</b>\n\n📋 <b>Subscription URL:</b>\n<code>%s</code>\n\n<i>Copy this link to your VPN client or scan the QR code below</i>", username, subURL), h.createUserActionKeyboard())
	if err != nil {
		return err
	}

	// Send QR code
	return h.sendQRCode(c, subURL)
}

// handleConfirmDelete handles the Delete action
func (h *AdminHandler) handleConfirmDelete(c telebot.Context, username string) error {
	// Установить состояние подтверждения удаления
	err := h.stateService.WithConversationState(c.Sender().ID, models.AwaitConfirmMemberDeletion)
	if err != nil {
		h.logger.Errorf("Failed to set state: %v", err)
		return err
	}
	// Показать клавиатуру подтверждения
	markup := h.createConfirmKeyboard()
	return h.sendTextMessage(c, fmt.Sprintf("🗑️ <b>Confirm User Deletion</b>\n\n⚠️ You are about to permanently delete user <b>%s</b>\n\n<b>This action will:</b>\n• Remove user from all server configurations\n• Delete all associated data\n• Cannot be undone\n\nAre you absolutely sure?", username), markup)
}

// processConfirmDeletion processes the deletion confirmation
func (h *AdminHandler) processConfirmDeletion(ctx context.Context, c telebot.Context) error {
	// Get confirmation from message
	confirmation := c.Text()

	// Check for return to main menu
	if h.getButtonCommand(confirmation) == commands.ReturnToMainMenu {
		return h.handleStart(ctx, c)
	}

	// Check if user confirmed
	if h.getButtonCommand(confirmation) != commands.Confirm {
		return h.sendTextMessage(c, "❌ <b>Invalid Selection</b>\n\nPlease click Confirm to proceed with deletion or use the Return button to cancel.", h.createConfirmKeyboard())
	}

	// Get user state to get the username we want to delete
	userState, err := h.stateService.GetState(c.Sender().ID)
	if err != nil {
		h.logger.Errorf("Failed to get user state: %v", err)
		return err
	}

	if userState.Payload == nil {
		return h.sendTextMessage(c, "❌ <b>Session Error</b>\n\nUser data was lost. Please start the deletion process again.", h.createReturnKeyboard())
	}

	username := *userState.Payload

	// Send loading message
	loadingMsg, _ := h.sendTextMessageWithReturn(c, fmt.Sprintf("⏳ <b>Deleting User...</b>\n\nRemoving user '%s' from all server configurations. Please wait...", username), nil)

	// Delete client using email
	err = h.xrayService.RemoveClients(ctx, []string{username})
	// Delete loading message
	if loadingMsg != nil {
		c.Bot().Delete(loadingMsg)
	}

	if err != nil {
		h.logger.Errorf("Failed to delete client: %v", err)
		return h.sendTextMessage(c, fmt.Sprintf("❌ <b>Deletion Failed</b>\n\nCouldn't delete user '%s'. Please try again or contact administrator.\n\n<b>Error:</b> %v", username, err), h.createReturnKeyboard())
	}

	return h.sendTextMessage(c, fmt.Sprintf("✅ <b>User Deleted Successfully</b>\n\n🗑️ User '%s' has been permanently removed from all server configurations.", username), h.createReturnKeyboard())
}

// showMembersWithSort показывает список пользователей с указанной сортировкой
func (h *AdminHandler) showMembersWithSort(ctx context.Context, c telebot.Context, sortType models.SortType, actionType string) error {
	// Get all members with detailed info
	members, err := h.xrayService.GetAllMembersWithInfo(ctx, sortType)
	if err != nil {
		h.logger.Errorf("Failed to get members with info: %v", err)
		return h.sendTextMessage(c, "❌ <b>Connection Error</b>\n\nCouldn't retrieve user list. Please check your server connection and try again.", h.createReturnKeyboard())
	}

	if len(members) == 0 {
		message := "📭 <b>No Users Found</b>\n\nThere are no users in the system yet."
		if actionType == "edit" {
			message += " Use <b>Add Member</b> to create your first user."
		}
		return h.sendTextMessage(c, message, h.createReturnKeyboard())
	}

	// Create keyboard with member names and additional info
	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}

	var rows []telebot.Row
	for _, member := range members {
		// Format button text with additional info based on sort type
		buttonText := h.formatMemberButtonText(member, sortType)
		rows = append(rows, telebot.Row{telebot.Btn{Text: buttonText}})
	}

	// Add return button
	rows = append(rows, telebot.Row{telebot.Btn{Text: "↩️ " + commands.ReturnToMainMenu}})

	markup.Reply(rows...)

	// Set appropriate state
	var nextState models.ConversationState
	var messageText string

	if actionType == "edit" {
		nextState = models.AwaitSelectUserName
		messageText = "✏️ <b>Edit User</b>\n\n👥 Select a user to manage:"
	} else if actionType == "delete" {
		nextState = models.AwaitConfirmMemberDeletion
		messageText = "🗑️ <b>Delete User</b>\n\n⚠️ Select a user to permanently delete:"
	}

	err = h.stateService.WithConversationState(c.Sender().ID, nextState)
	if err != nil {
		h.logger.Errorf("Failed to set state: %v", err)
		return err
	}

	return h.sendTextMessage(c, messageText, markup)
}

// formatMemberButtonText форматирует текст кнопки пользователя с дополнительной информацией
func (h *AdminHandler) formatMemberButtonText(member models.MemberInfo, sortType models.SortType) string {
	baseText := member.BaseUsername

	switch sortType {
	case models.SortByCreationOrder:
		return baseText // По дате добавления показываем только имя
	case models.SortByExpiryDate:
		return fmt.Sprintf("%s (%s)", baseText, member.GetExpiryStatus())
	case models.SortByTrafficTotal:
		if member.TotalTraffic > 0 {
			totalGB := float64(member.TotalTraffic) / (1024 * 1024 * 1024)
			return fmt.Sprintf("%s (%.1f GB)", baseText, totalGB)
		}
		return fmt.Sprintf("%s (0 GB)", baseText)
	case models.SortByStatus:
		status := "❌"
		if member.Enable {
			status = "✅"
		}
		return fmt.Sprintf("%s %s", status, baseText)
	default:
		return baseText
	}
}
