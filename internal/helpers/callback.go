package helpers

import "strings"

// CallbackCommand extracts the command from inline-button callback data.
// telebot encodes it as "\f<unique>" or "\f<unique>|<payload>"; only handlers
// registered per unique get it pre-split, and this bot routes every callback
// through one entry point, so it splits the data itself.
func CallbackCommand(data string) string {
	data = strings.TrimPrefix(data, "\f")
	if idx := strings.Index(data, "|"); idx >= 0 {
		data = data[:idx]
	}
	return strings.TrimSpace(data)
}
