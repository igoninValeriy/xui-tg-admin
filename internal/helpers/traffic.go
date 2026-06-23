package helpers

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"xui-tg-admin/internal/models"
)

// nearExpiryWindow is how soon before expiry a user is flagged as "expiring".
const nearExpiryWindow = 7 * 24 * time.Hour

// UserTraffic is the aggregated traffic of a single user (all their clients
// across every inbound collapsed into one entry).
type UserTraffic struct {
	Name       string
	Up         int64
	Down       int64
	Online     bool
	Enabled    bool
	ExpiryTime int64 // Unix ms, 0 = unlimited
}

// Total returns the combined up+down traffic in bytes.
func (u UserTraffic) Total() int64 { return u.Up + u.Down }

// ExpiringSoon reports whether the user's expiry falls within nearExpiryWindow.
func (u UserTraffic) ExpiringSoon(now time.Time) bool {
	if u.ExpiryTime == 0 {
		return false
	}
	exp := time.UnixMilli(u.ExpiryTime)
	return exp.After(now) && exp.Before(now.Add(nearExpiryWindow))
}

// InboundTraffic is the aggregated traffic of a single inbound (summed over all
// its clients), identified by its remark.
type InboundTraffic struct {
	Name string
	Up   int64
	Down int64
}

// Total returns the combined up+down traffic in bytes.
func (i InboundTraffic) Total() int64 { return i.Up + i.Down }

// TrafficReport is the fully aggregated, sorted view used by both the text
// fallback and the rendered image. Pure data, no formatting.
type TrafficReport struct {
	Users       []UserTraffic    // sorted by total traffic desc, then name asc
	Inbounds    []InboundTraffic // sorted by total traffic desc, then name asc
	TotalUp     int64
	TotalDown   int64
	OnlineCount int
}

// AggregateTraffic collapses all client stats into per-user totals, grouping a
// user's clients by SubID (falling back to the base username), and returns a
// sorted report. Pure function — no I/O.
func AggregateTraffic(inbounds []models.Inbound, onlineUsers []string) TrafficReport {
	onlineSet := make(map[string]bool, len(onlineUsers))
	for _, u := range onlineUsers {
		onlineSet[ExtractBaseUsername(u)] = true
	}

	emailToSubID := BuildEmailToSubID(inbounds)
	byGroup := make(map[string]*UserTraffic)
	byInbound := make(map[string]*InboundTraffic)

	for _, inbound := range inbounds {
		for _, cs := range inbound.ClientStats {
			key := UserGroupKey(cs.Email, emailToSubID)
			u := byGroup[key]
			if u == nil {
				u = &UserTraffic{Name: ExtractBaseUsername(cs.Email)}
				byGroup[key] = u
			}
			u.Up += cs.Up
			u.Down += cs.Down
			if cs.Enable {
				u.Enabled = true
			}
			if cs.ExpiryTime > u.ExpiryTime {
				u.ExpiryTime = cs.ExpiryTime
			}

			in := byInbound[inbound.Remark]
			if in == nil {
				in = &InboundTraffic{Name: inbound.Remark}
				byInbound[inbound.Remark] = in
			}
			in.Up += cs.Up
			in.Down += cs.Down
		}
	}

	report := TrafficReport{Users: make([]UserTraffic, 0, len(byGroup))}
	for _, u := range byGroup {
		u.Online = onlineSet[u.Name]
		report.Users = append(report.Users, *u)
		report.TotalUp += u.Up
		report.TotalDown += u.Down
		if u.Online {
			report.OnlineCount++
		}
	}

	sort.Slice(report.Users, func(i, j int) bool {
		ti, tj := report.Users[i].Total(), report.Users[j].Total()
		if ti == tj {
			return report.Users[i].Name < report.Users[j].Name
		}
		return ti > tj
	})

	report.Inbounds = make([]InboundTraffic, 0, len(byInbound))
	for _, in := range byInbound {
		report.Inbounds = append(report.Inbounds, *in)
	}
	sort.Slice(report.Inbounds, func(i, j int) bool {
		ti, tj := report.Inbounds[i].Total(), report.Inbounds[j].Total()
		if ti == tj {
			return report.Inbounds[i].Name < report.Inbounds[j].Name
		}
		return ti > tj
	})

	return report
}

// FormatBytes renders a byte count with an adaptive unit (B/KB/MB/GB/TB).
func FormatBytes(b int64) string {
	const (
		kb = 1 << 10
		mb = 1 << 20
		gb = 1 << 30
		tb = 1 << 40
	)
	f := float64(b)
	switch {
	case b >= tb:
		return fmt.Sprintf("%.2f TB", f/tb)
	case b >= gb:
		return fmt.Sprintf("%.2f GB", f/gb)
	case b >= mb:
		return fmt.Sprintf("%.1f MB", f/mb)
	case b >= kb:
		return fmt.Sprintf("%.1f KB", f/kb)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

const (
	// maxTextRows caps how many user rows the text table prints, keeping the
	// message under Telegram's 4096-char limit.
	maxTextRows = 60
	// tableNameW is the width of the name column. The whole row is
	// tableNameW + 14 chars (name + " " + 6 + " " + 6) — narrow enough that
	// Telegram does not wrap the <pre> block on a phone.
	tableNameW = 13
)

// FormatTrafficText renders the report as a monospace <pre> table. Columns are
// aligned by padding the raw (pre-escape) name to a fixed width; HTML escaping
// happens afterwards so entities like &lt; never throw the alignment off.
func FormatTrafficText(report TrafficReport, _ time.Time) string {
	if len(report.Users) == 0 {
		return "📭 <b>No Users Found</b>\n\nThere are no users in the system yet."
	}

	var sb strings.Builder
	sb.WriteString("📊 <b>Traffic usage</b> · since last reset\n\n")

	sb.WriteString("<pre>\n")
	sb.WriteString(tableRow("User", "Down", "Up"))
	sb.WriteString(tableRule())

	rows, extra := report.Users, 0
	if len(rows) > maxTextRows {
		extra = len(rows) - maxTextRows
		rows = rows[:maxTextRows]
	}
	for _, u := range rows {
		sb.WriteString(tableRow(u.Name, FormatBytesShort(u.Down), FormatBytesShort(u.Up)))
	}

	sb.WriteString(tableRule())
	sb.WriteString(tableRow("TOTAL", FormatBytesShort(report.TotalDown), FormatBytesShort(report.TotalUp)))
	sb.WriteString("</pre>\n")

	if extra > 0 {
		sb.WriteString(fmt.Sprintf("…and %d more\n", extra))
	}
	sb.WriteString(fmt.Sprintf("👥 %d users · 🟢 %d online\n", len(report.Users), report.OnlineCount))

	if len(report.Inbounds) > 0 {
		sb.WriteString("\n📡 <b>By inbound</b>\n<pre>\n")
		for _, in := range report.Inbounds {
			sb.WriteString(tableRow(in.Name, FormatBytesShort(in.Down), FormatBytesShort(in.Up)))
		}
		sb.WriteString("</pre>")
	}

	return sb.String()
}

// tableRow formats one aligned <pre> row: name left-padded to tableNameW,
// then two 6-wide right-aligned number columns.
func tableRow(name, down, up string) string {
	return fmt.Sprintf("%s %6s %6s\n", escapeHTML(padName(name, tableNameW)), down, up)
}

// tableRule returns a horizontal rule the exact width of a table row.
func tableRule() string {
	return strings.Repeat("─", tableNameW+14) + "\n"
}

// padName pads name to w visual columns, truncating with an ellipsis if longer.
func padName(name string, w int) string {
	r := []rune(name)
	if len(r) > w {
		return string(r[:w-1]) + "…"
	}
	return name + strings.Repeat(" ", w-len(r))
}

// FormatBytesShort renders a byte count in at most ~5 chars (e.g. 98.2G, 734M),
// for compact table columns.
func FormatBytesShort(b int64) string {
	const (
		kb = 1 << 10
		mb = 1 << 20
		gb = 1 << 30
		tb = 1 << 40
	)
	f := float64(b)
	switch {
	case b >= tb:
		return fmt.Sprintf("%.1fT", f/tb)
	case b >= 100*gb:
		return fmt.Sprintf("%.0fG", f/gb)
	case b >= gb:
		return fmt.Sprintf("%.1fG", f/gb)
	case b >= 100*mb:
		return fmt.Sprintf("%.0fM", f/mb)
	case b >= mb:
		return fmt.Sprintf("%.1fM", f/mb)
	case b >= kb:
		return fmt.Sprintf("%.0fK", f/kb)
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// escapeHTML escapes the characters that are special in Telegram HTML mode.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
