package helpers

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"xui-tg-admin/internal/constants"
	"xui-tg-admin/internal/models"
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

// FormatCompactTrafficReport formats a compact and beautiful traffic report for X-Ray users
func FormatCompactTrafficReport(inbounds []models.Inbound, onlineUsers []string) string {
	if len(inbounds) == 0 {
		return "📭 <b>No Users Found</b>\n\nThere are no users in the system yet."
	}

	// Create a set of online users for quick lookup
	onlineSet := make(map[string]bool)
	for _, user := range onlineUsers {
		// Extract base username from online user email
		baseUser := ExtractBaseUsername(user)
		onlineSet[baseUser] = true
	}

	// Map each email to its SubID so a user's clients across different protocols
	// group together, regardless of how the username is spelled.
	emailToSubID := BuildEmailToSubID(inbounds)

	// Aggregate user data by group key (SubID, falling back to base username)
	userSummary := make(map[string]*UserTrafficSummary)

	for _, inbound := range inbounds {
		for _, clientStat := range inbound.ClientStats {
			groupKey := UserGroupKey(clientStat.Email, emailToSubID)

			if userSummary[groupKey] == nil {
				userSummary[groupKey] = &UserTrafficSummary{
					BaseUsername: ExtractBaseUsername(clientStat.Email),
					TotalUp:      0,
					TotalDown:    0,
					Enable:       clientStat.Enable,
					ExpiryTime:   clientStat.ExpiryTime,
					InboundStats: make(map[string]*InboundTrafficStats),
				}
			}

			summary := userSummary[groupKey]
			summary.TotalUp += clientStat.Up
			summary.TotalDown += clientStat.Down

			// Keep enabled status if any client is enabled
			if clientStat.Enable {
				summary.Enable = true
			}

			// Use the latest expiry time
			if clientStat.ExpiryTime > summary.ExpiryTime {
				summary.ExpiryTime = clientStat.ExpiryTime
			}

			// Track stats per inbound
			if summary.InboundStats[inbound.Remark] == nil {
				summary.InboundStats[inbound.Remark] = &InboundTrafficStats{
					Down: 0,
					Up:   0,
				}
			}
			summary.InboundStats[inbound.Remark].Down += clientStat.Down
			summary.InboundStats[inbound.Remark].Up += clientStat.Up
		}
	}

	if len(userSummary) == 0 {
		return "📭 <b>No Active Users</b>\n\nNo user traffic data available."
	}

	// Convert to slice for sorting
	var users []*UserTrafficSummary
	for _, summary := range userSummary {
		users = append(users, summary)
	}

	// Sort users by total traffic (descending), then by name (ascending) for ties
	sort.Slice(users, func(i, j int) bool {
		totalI := users[i].TotalUp + users[i].TotalDown
		totalJ := users[j].TotalUp + users[j].TotalDown

		if totalI == totalJ {
			// If traffic is equal, sort by username alphabetically
			return users[i].BaseUsername < users[j].BaseUsername
		}

		// Sort by total traffic (descending)
		return totalI > totalJ
	})

	// Calculate totals
	var grandTotalUp, grandTotalDown int64

	// Prepare all report lines
	var reportLines []TrafficReportLine

	// Add user lines
	for _, summary := range users {
		grandTotalUp += summary.TotalUp
		grandTotalDown += summary.TotalDown

		// Determine online status
		statusIcon := "🔴"
		if onlineSet[summary.BaseUsername] {
			statusIcon = "🟢"
		}

		// Show the full base username (only the inbound-number suffix is stripped)
		displayName := summary.BaseUsername

		// Convert traffic to GB with 2 decimal places
		downGB := float64(summary.TotalDown) / constants.BytesInGB
		upGB := float64(summary.TotalUp) / constants.BytesInGB

		// Add expiry info if set
		expiryInfo := ""
		if summary.ExpiryTime > 0 {
			expiryDate := time.Unix(summary.ExpiryTime/1000, 0)
			expiryInfo = fmt.Sprintf(" (until %s)", expiryDate.Format("02.01.06"))
		}

		reportLines = append(reportLines, TrafficReportLine{
			StatusIcon:  statusIcon,
			DisplayName: displayName,
			DownGB:      downGB,
			UpGB:        upGB,
			ExtraInfo:   expiryInfo,
			IsTotal:     false,
		})
	}

	// Add separator line
	reportLines = append(reportLines, TrafficReportLine{
		StatusIcon:  "",
		DisplayName: "────────────────",
		DownGB:      0,
		UpGB:        0,
		ExtraInfo:   "",
		IsSeparator: true,
	})

	// Add grand total line
	grandTotalDownGB := float64(grandTotalDown) / constants.BytesInGB
	grandTotalUpGB := float64(grandTotalUp) / constants.BytesInGB

	reportLines = append(reportLines, TrafficReportLine{
		StatusIcon:  "📊",
		DisplayName: "Total",
		DownGB:      grandTotalDownGB,
		UpGB:        grandTotalUpGB,
		ExtraInfo:   "",
		IsTotal:     true,
	})

	// Add per-inbound breakdown
	inboundTotals := make(map[string]*InboundTrafficStats)
	for _, summary := range users {
		for inboundName, stats := range summary.InboundStats {
			if inboundTotals[inboundName] == nil {
				inboundTotals[inboundName] = &InboundTrafficStats{Down: 0, Up: 0}
			}
			inboundTotals[inboundName].Down += stats.Down
			inboundTotals[inboundName].Up += stats.Up
		}
	}

	// Sort inbound names for consistent output
	var inboundNames []string
	for name := range inboundTotals {
		inboundNames = append(inboundNames, name)
	}
	sort.Strings(inboundNames)

	// Add inbound breakdown lines
	for _, inboundName := range inboundNames {
		stats := inboundTotals[inboundName]
		downGB := float64(stats.Down) / constants.BytesInGB
		upGB := float64(stats.Up) / constants.BytesInGB

		reportLines = append(reportLines, TrafficReportLine{
			StatusIcon:  "📡",
			DisplayName: inboundName,
			DownGB:      downGB,
			UpGB:        upGB,
			ExtraInfo:   "",
			IsInbound:   true,
		})
	}

	// Format all lines with consistent alignment
	var sb strings.Builder
	sb.WriteString("<b>📊 Traffic Usage Report</b>\n\n")
	sb.WriteString("<pre>")

	for _, line := range reportLines {
		sb.WriteString(formatTrafficReportLine(line) + "\n")
	}

	sb.WriteString("</pre>")

	return sb.String()
}

// TrafficReportLine represents a single line in the traffic report
type TrafficReportLine struct {
	StatusIcon  string  // Status icon (🟢, 🔴, 📊, 📡, etc.)
	DisplayName string  // Name to display
	DownGB      float64 // Download traffic in GB
	UpGB        float64 // Upload traffic in GB
	ExtraInfo   string  // Additional info (expiry, etc.)
	IsTotal     bool    // Whether this is a total line
	IsInbound   bool    // Whether this is an inbound line
	IsSeparator bool    // Whether this is a separator line
}

// formatTrafficReportLine formats a single line of the traffic report with consistent alignment
func formatTrafficReportLine(line TrafficReportLine) string {
	const nameWidth = 16
	const trafficWidth = 8

	// Handle separator line
	if line.IsSeparator {
		return line.DisplayName + "─────────────────"
	}

	// Display the full name; %-*s pads short names but never truncates long ones,
	// so usernames are shown in full even if they exceed the nominal column width.
	displayName := line.DisplayName

	// Format traffic values
	var trafficStr string
	if line.IsTotal || line.IsInbound {
		// For totals and inbounds, show traffic in bold-like format
		trafficStr = fmt.Sprintf("%*.2f GB ⬇ %*.2f GB ⬆",
			trafficWidth, line.DownGB, trafficWidth-1, line.UpGB)
	} else {
		// For regular users, standard format
		trafficStr = fmt.Sprintf("%*.2f GB ⬇ %*.2f GB ⬆",
			trafficWidth, line.DownGB, trafficWidth-1, line.UpGB)
	}

	// Combine all parts
	if line.StatusIcon != "" {
		return fmt.Sprintf("%s %-*s %s%s",
			line.StatusIcon, nameWidth, displayName, trafficStr, line.ExtraInfo)
	} else {
		return fmt.Sprintf("  %-*s %s%s",
			nameWidth, displayName, trafficStr, line.ExtraInfo)
	}
}

// UserTrafficSummary represents aggregated traffic data for a user
type UserTrafficSummary struct {
	BaseUsername string
	TotalUp      int64
	TotalDown    int64
	Enable       bool
	ExpiryTime   int64
	InboundStats map[string]*InboundTrafficStats
}

// InboundTrafficStats represents traffic stats for a specific inbound
type InboundTrafficStats struct {
	Down int64
	Up   int64
}
