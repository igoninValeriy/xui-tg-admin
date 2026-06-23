package validation

import (
	"fmt"
	"strconv"
	"xui-tg-admin/internal/constants"
)

// ValidateUsername rejects only what would actually break downstream use. The
// name is interpolated verbatim into X-UI API URLs (e.g. .../delClient/<name>-N)
// and used as a client label, so it must consist of URL-safe "unreserved"
// characters (RFC 3986): letters, digits, '-', '.', '_', '~'. Everything else
// (spaces, '/', '?', '#', '%', control chars, non-ASCII) is rejected because it
// would corrupt the request URL.
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if len(username) > constants.MaxUsernameLength {
		return fmt.Errorf("username is too long — use at most %d characters", constants.MaxUsernameLength)
	}

	for _, r := range username {
		if !isURLSafeUsernameChar(r) {
			return fmt.Errorf("username can't contain %q — allowed: letters, digits, and - . _ ~", r)
		}
	}

	return nil
}

// ValidateDuration validates and parses a duration string
func ValidateDuration(durationStr string) (int, error) {
	days, err := strconv.Atoi(durationStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: must be a number")
	}

	if days < constants.MinDurationDays {
		return 0, fmt.Errorf("duration must be at least %d day", constants.MinDurationDays)
	}

	if days > constants.MaxDurationDays {
		return 0, fmt.Errorf("duration cannot exceed %d days", constants.MaxDurationDays)
	}

	return days, nil
}

// isURLSafeUsernameChar reports whether r is an RFC 3986 "unreserved" character —
// safe verbatim in a URL path segment and as an X-UI label. Dashes are fine:
// ExtractBaseUsername only strips the rightmost "-<digits>" (the inbound suffix),
// so dashes inside a name are preserved.
func isURLSafeUsernameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '.' || r == '_' || r == '~'
}
