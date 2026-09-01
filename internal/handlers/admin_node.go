package handlers

import (
	"context"
	"strings"
	"time"

	telebot "gopkg.in/telebot.v3"

	"xui-tg-admin/internal/helpers"
	"xui-tg-admin/internal/permissions"
)

// handleNodeStatus shows the state of the host the bot shares with the VPN
// stack. Every value comes from the tool that already owns it on that host, so
// there is one source of truth; a probe that fails is reported as missing
// rather than guessed at.
func (h *AdminHandler) handleNodeStatus(ctx context.Context, c telebot.Context) error {
	status := h.nodeService.Status(ctx)
	return h.sendTextMessage(c, helpers.FormatNodeStatus(status, time.Now()), h.createNodeRefreshKeyboard())
}

// handleNodeStatusRefresh redraws the node status message in place in response
// to the inline Refresh button.
func (h *AdminHandler) handleNodeStatusRefresh(ctx context.Context, c telebot.Context) error {
	status := h.nodeService.Status(ctx)

	if err := c.Respond(&telebot.CallbackResponse{Text: "Refreshed"}); err != nil {
		h.logger.Errorf("Failed to answer callback: %v", err)
	}

	opts := &telebot.SendOptions{
		ParseMode:   telebot.ModeHTML,
		ReplyMarkup: h.createNodeRefreshKeyboard(),
	}

	err := c.Edit(helpers.FormatNodeStatus(status, time.Now()), opts)
	// Telegram rejects an edit that changes nothing; with an unchanged host and
	// the same timestamp that is a no-op, not a failure.
	if err != nil && strings.Contains(err.Error(), "message is not modified") {
		return nil
	}
	if err != nil {
		h.logger.Errorf("Failed to edit node status message: %v", err)
	}
	return err
}

// handleSubscriptions shows the output of the host's subscription statistics
// script verbatim, in a monospaced block.
func (h *AdminHandler) handleSubscriptions(ctx context.Context, c telebot.Context) error {
	mainKeyboard := h.createMainKeyboard(permissions.Admin)

	loadingMsg, _ := h.sendTextMessageWithReturn(c, "⏳ <b>Reading subscription statistics...</b>\n\nThe script queries the subscription store, so this can take a while.", nil)
	output, err := h.nodeService.SubscriptionStats(ctx)
	if loadingMsg != nil {
		if delErr := c.Bot().Delete(loadingMsg); delErr != nil {
			h.logger.Errorf("Failed to delete loading message: %v", delErr)
		}
	}

	if err != nil {
		h.logger.Errorf("Failed to collect subscription stats: %v", err)
		return h.sendTextMessage(c, "📡 <b>Subscriptions</b>\n\n❌ No data: the statistics script could not be run on this host.", mainKeyboard)
	}

	return h.sendTextMessage(c, helpers.FormatSubscriptionStats(output), mainKeyboard)
}

// handleSubscriptionQR sends a QR code of the user's subscription URL. It
// deliberately encodes the subscription link and not a config link: a config
// imported by hand never updates itself, and people have been bitten by that.
func (h *AdminHandler) handleSubscriptionQR(ctx context.Context, c telebot.Context, username string) error {
	clients, err := h.xrayService.FindMemberClients(ctx, username)
	if err != nil {
		h.logger.Errorf("Failed to look up client %s: %v", username, err)
		return h.sendTextMessage(c, "❌ <b>Connection Error</b>\n\nCouldn't read the client list. Please try again.", h.createUserActionKeyboard())
	}
	if len(clients) == 0 {
		return h.sendTextMessage(c, "❌ <b>User Not Found</b>\n\nNo configuration found for user '"+username+"'.", h.createUserActionKeyboard())
	}

	subID := ""
	for _, record := range clients {
		if record.SubID != "" {
			subID = record.SubID
			break
		}
	}

	subURL := h.xrayService.SubscriptionURL(subID)
	if subURL == "" {
		return h.sendTextMessage(c, "❌ <b>No Subscription</b>\n\nThis user has no subscription ID, so there is no subscription link to encode.", h.createUserActionKeyboard())
	}

	caption := "📡 <b>Subscription for " + username + "</b>\n\n" +
		"⚠️ <b>Add this in the client as a SUBSCRIPTION, not as a config.</b>\n\n" +
		"A link imported as a plain config or server is a one-off snapshot: it never refreshes, " +
		"and it stops working the moment the server side changes. Added as a subscription, the " +
		"client keeps itself up to date."

	return h.sendQRCodeWithCaption(c, subURL, caption, h.createUserActionKeyboard())
}
