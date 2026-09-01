package handlers

import (
	"context"
	"time"

	telebot "gopkg.in/telebot.v3"

	"xui-tg-admin/internal/commands"
	"xui-tg-admin/internal/helpers"
	"xui-tg-admin/internal/models"
	"xui-tg-admin/internal/permissions"
	"xui-tg-admin/internal/validation"
)

// ClientCreationParams holds parameters for client creation
type ClientCreationParams struct {
	BaseUsername string
	DurationStr  string
	ExpiryTime   int64
	SubID        string
	SenderID     int64
}

// createClient creates one client attached to every enabled inbound. Panel
// 3.7.0 treats a client as a first-class entity with an inbound list, so the
// old "one client per inbound, named username-1, username-2, …" scheme is gone:
// the user is a single record addressed by its email everywhere.
func (h *AdminHandler) createClient(ctx context.Context, params ClientCreationParams, inboundIDs []int) error {
	client := models.Client{
		Email:      params.BaseUsername,
		Enable:     true,
		TotalGB:    0, // Unlimited traffic
		LimitIP:    0, // No IP limit
		ExpiryTime: params.ExpiryTime,
		TgID:       params.SenderID,
		SubID:      params.SubID,
	}

	return h.xrayService.AddClient(ctx, client, inboundIDs)
}

// sendSubscriptionInfo reports the created user, its share links and its
// subscription URL, then returns to the main menu.
func (h *AdminHandler) sendSubscriptionInfo(ctx context.Context, c telebot.Context, params ClientCreationParams) error {
	// The panel builds the share links itself, so the bot never has to assemble
	// a vless:// URL from stream settings.
	links, err := h.xrayService.GetClientLinks(ctx, params.BaseUsername)
	if err != nil {
		h.logger.Errorf("Failed to get client links for %s: %v", params.BaseUsername, err)
	}

	subURL := h.xrayService.SubscriptionURL(params.SubID)

	subscriptionInfo := helpers.FormatSubscriptionInfo(
		params.BaseUsername,
		params.DurationStr,
		params.ExpiryTime,
		subURL,
		links,
	)

	if err := h.sendTextMessage(c, subscriptionInfo, nil); err != nil {
		return err
	}

	// The QR encodes the direct share link: it works the moment the client is
	// created, whereas the subscription URL only resolves once the subscription
	// front end has picked the new client up.
	if qrTarget := helpers.PreferredQRTarget(links, subURL); qrTarget != "" {
		if err := h.sendTextMessage(c, "QR code for the configuration:", nil); err != nil {
			h.logger.Errorf("Failed to send QR code message: %v", err)
		} else if err := h.sendQRCode(c, qrTarget); err != nil {
			h.logger.Errorf("Failed to send QR code: %v", err)
		}
	}

	// Clear user state and return to main menu
	if err := h.stateService.ClearState(c.Sender().ID); err != nil {
		h.logger.Errorf("Failed to clear user state: %v", err)
	}

	// Show main menu
	markup := h.createMainKeyboard(permissions.Admin)
	return h.sendTextMessage(c, "🎉 <b>User Created Successfully!</b>\n\nThe new user is ready to connect to the VPN.", markup)
}

// calculateExpiryTime calculates expiry time based on duration
func calculateExpiryTime(durationStr string) (int64, error) {
	if durationStr == commands.Infinite {
		return 0, nil
	}

	days, err := validation.ValidateDuration(durationStr)
	if err != nil {
		return 0, err
	}

	return time.Now().Add(time.Duration(days) * 24 * time.Hour).UnixMilli(), nil
}
