package helpers

import (
	"fmt"
	"strings"
	"time"

	"xui-tg-admin/internal/constants"
)

// FormatSubscriptionInfo formats subscription information for a single user
func FormatSubscriptionInfo(baseUsername string, durationStr string, expiryTime int64, createdEmails []string, commonSubId string, addErrors []string, subURLPrefix string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Client added successfully!\n\nBase username: %s\n", baseUsername))

	if expiryTime == 0 {
		sb.WriteString("Duration: ∞ (infinite)\n")
	} else {
		sb.WriteString(fmt.Sprintf("Duration: %s days\nExpiry: %s\n",
			durationStr,
			time.Unix(expiryTime/1000, 0).Format(constants.DateFormat)))
	}

	sb.WriteString("Traffic limit: Unlimited\n")
	sb.WriteString("\nCreated accounts:\n")
	for _, email := range createdEmails {
		sb.WriteString(fmt.Sprintf("\n- %s", email))
	}

	if len(createdEmails) > 0 {
		subURL := fmt.Sprintf("%s%s?name=%s", subURLPrefix, commonSubId, commonSubId)
		sb.WriteString(fmt.Sprintf("\n\nLink to connect: %s", subURL))
	}

	if len(addErrors) > 0 {
		sb.WriteString(fmt.Sprintf("\n\nWarning: Failed to add to some inbounds:\n%s\n", strings.Join(addErrors, "\n")))
	}

	return sb.String()
}
