package handlers

import (
	"bytes"

	"github.com/sirupsen/logrus"
	telebot "gopkg.in/telebot.v3"

	"xui-tg-admin/internal/commands"
	"xui-tg-admin/internal/config"
	"xui-tg-admin/internal/permissions"
	"xui-tg-admin/internal/services"
)

// BaseHandler provides common functionality for all handlers
type BaseHandler struct {
	xrayService  *services.XrayService
	stateService *services.UserStateService
	qrService    *services.QRService
	config       *config.Config
	logger       *logrus.Logger
}

// NewBaseHandler creates a new base handler
func NewBaseHandler(
	xrayService *services.XrayService,
	stateService *services.UserStateService,
	qrService *services.QRService,
	config *config.Config,
	logger *logrus.Logger,
) BaseHandler {
	return BaseHandler{
		xrayService:  xrayService,
		stateService: stateService,
		qrService:    qrService,
		config:       config,
		logger:       logger,
	}
}

// CanHandle checks if the handler can handle the given access type
func (h *BaseHandler) CanHandle(accessType permissions.AccessType) bool {
	// Base handler can't handle any access type directly
	return false
}

// sendTextMessage sends a text message with optional markup
func (h *BaseHandler) sendTextMessage(c telebot.Context, text string, markup *telebot.ReplyMarkup) error {
	opts := &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
	}

	if markup != nil {
		opts.ReplyMarkup = markup
	}

	_, err := c.Bot().Send(c.Recipient(), text, opts)
	if err != nil {
		h.logger.Errorf("Failed to send message: %v", err)
	}
	return err
}

// sendTextMessageWithReturn sends a text message and returns the message for deletion
func (h *BaseHandler) sendTextMessageWithReturn(c telebot.Context, text string, markup *telebot.ReplyMarkup) (*telebot.Message, error) {
	opts := &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
	}

	if markup != nil {
		opts.ReplyMarkup = markup
	}

	msg, err := c.Bot().Send(c.Recipient(), text, opts)
	if err != nil {
		h.logger.Errorf("Failed to send message: %v", err)
	}
	return msg, err
}

// sendQRCode sends a QR code for the given URL
func (h *BaseHandler) sendQRCode(c telebot.Context, url string) error {
	// Generate QR code
	qrBytes, err := h.qrService.GenerateQR(url)
	if err != nil {
		h.logger.Errorf("Failed to generate QR code: %v", err)
		return err
	}

	// Create photo from bytes
	reader := bytes.NewReader(qrBytes)
	photo := &telebot.Photo{File: telebot.FromReader(reader)}

	// Send photo
	_, err = c.Bot().Send(c.Recipient(), photo)
	if err != nil {
		h.logger.Errorf("Failed to send QR code: %v", err)
	}
	return err
}

// sendPhotoBytes sends a raw image (e.g. a rendered PNG) as a photo, optionally
// with a reply keyboard.
func (h *BaseHandler) sendPhotoBytes(c telebot.Context, img []byte, markup *telebot.ReplyMarkup) error {
	photo := &telebot.Photo{File: telebot.FromReader(bytes.NewReader(img))}
	var opts []interface{}
	if markup != nil {
		opts = append(opts, markup)
	}
	_, err := c.Bot().Send(c.Recipient(), photo, opts...)
	if err != nil {
		h.logger.Errorf("Failed to send photo: %v", err)
	}
	return err
}

// createMainKeyboard creates the main keyboard for the given access type
func (h *BaseHandler) createMainKeyboard(accessType permissions.AccessType) *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}

	var rows []telebot.Row

	switch accessType {
	case permissions.Admin:
		rows = []telebot.Row{
			{
				telebot.Btn{Text: "👤 " + commands.AddMember},
				telebot.Btn{Text: "🟢 " + commands.OnlineMembers},
			},
			{
				telebot.Btn{Text: "✏️ " + commands.EditMember},
				telebot.Btn{Text: "📈 " + commands.DetailedUsage},
			},
			{
				telebot.Btn{Text: "🔄 " + commands.ResetNetworkUsage},
			},
		}
	}

	markup.Reply(rows...)
	return markup
}

// createUsageReportKeyboard creates the Detailed Usage sub-menu (Table / Photo / Back)
func (h *BaseHandler) createUsageReportKeyboard() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}

	markup.Reply(
		telebot.Row{
			telebot.Btn{Text: "📊 " + commands.UsageTable},
			telebot.Btn{Text: "🖼 " + commands.UsagePhoto},
		},
		telebot.Row{
			telebot.Btn{Text: "↩️ " + commands.ReturnToMainMenu},
		},
	)

	return markup
}

// createReturnKeyboard creates a keyboard with a return button
func (h *BaseHandler) createReturnKeyboard() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}

	markup.Reply(
		telebot.Row{
			telebot.Btn{Text: "↩️ " + commands.ReturnToMainMenu},
		},
	)

	return markup
}

// createConfirmKeyboard creates a keyboard for confirmation (Confirm / Return)
func (h *BaseHandler) createConfirmKeyboard() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}

	markup.Reply(
		telebot.Row{
			telebot.Btn{Text: "✅ " + commands.Confirm},
		},
		telebot.Row{
			telebot.Btn{Text: "↩️ " + commands.ReturnToMainMenu},
		},
	)

	return markup
}
