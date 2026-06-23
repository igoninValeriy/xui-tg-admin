package helpers

import (
	"encoding/json"

	"xui-tg-admin/internal/models"
)

// IsNumeric checks if a string contains only digits
func IsNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// BuildEmailToSubID parses inbound settings and maps each client email to its
// subscription ID. Clients of the same user across different protocols/inbounds
// share one SubID, so it is the reliable key for grouping a user together.
func BuildEmailToSubID(inbounds []models.Inbound) map[string]string {
	emailToSubID := make(map[string]string)
	for _, inbound := range inbounds {
		if inbound.Settings == "" {
			continue
		}
		var settings models.InboundSettings
		if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
			continue
		}
		for _, client := range settings.Clients {
			if client.SubID != "" {
				emailToSubID[client.Email] = client.SubID
			}
		}
	}
	return emailToSubID
}

// UserGroupKey returns the stable grouping key for a client email: its SubID when
// known, otherwise the base username (the email without the inbound-number
// suffix). This groups one user's clients across protocols without relying on
// name parsing, which would mangle usernames that contain dashes or underscores.
func UserGroupKey(email string, emailToSubID map[string]string) string {
	if subID, ok := emailToSubID[email]; ok && subID != "" {
		return subID
	}
	return ExtractBaseUsername(email)
}
