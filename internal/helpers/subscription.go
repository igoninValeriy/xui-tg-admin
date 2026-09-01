package helpers

import (
	"fmt"
	"strings"
	"time"

	"xui-tg-admin/internal/constants"
)

// FormatSubscriptionInfo renders the summary shown after a user is created:
// its validity, the panel's ready-made share links and the subscription URL.
func FormatSubscriptionInfo(
	baseUsername string,
	durationStr string,
	expiryTime int64,
	subURL string,
	links []string,
) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Client added successfully!\n\nUsername: %s\n", baseUsername))

	if expiryTime == 0 {
		sb.WriteString("Duration: ∞ (infinite)\n")
	} else {
		sb.WriteString(fmt.Sprintf("Duration: %s days\nExpiry: %s\n",
			durationStr,
			time.Unix(expiryTime/1000, 0).Format(constants.DateFormat)))
	}

	sb.WriteString("Traffic limit: Unlimited\n")

	if len(links) > 0 {
		sb.WriteString("\nConfiguration links:\n")
		for _, link := range links {
			sb.WriteString(fmt.Sprintf("\n%s\n", link))
		}
	} else {
		sb.WriteString("\nNo configuration link was returned by the panel.\n")
	}

	if subURL != "" {
		sb.WriteString(fmt.Sprintf("\nSubscription URL: %s\n", subURL))
		sb.WriteString("\nNote: the subscription is served by the external subscription front end, " +
			"so the URL starts working only after that front end has picked the new client up.\n")
	}

	return sb.String()
}

// PreferredQRTarget picks what the QR code should encode: the first share link
// if the panel returned one, otherwise the subscription URL. A share link works
// immediately, while a subscription URL depends on the front end being refreshed.
func PreferredQRTarget(links []string, subURL string) string {
	if len(links) > 0 {
		return links[0]
	}
	return subURL
}
