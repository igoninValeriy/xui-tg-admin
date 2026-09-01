package helpers

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"xui-tg-admin/internal/constants"
	"xui-tg-admin/internal/models"
)

// noData is what every field falls back to when its probe gave no answer. The
// screen never invents a value: a missing answer is shown as missing.
const noData = "no data"

// ansiPattern matches the CSI escape sequences a coloured CLI tool emits.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

// StripANSI removes ANSI escape sequences from command output so it can be put
// inside a Telegram <code> block verbatim.
func StripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

// PortIsListening reports whether the output of "ss -ltn" contains a listening
// socket on the given port, mirroring the `ss -ltn | grep ':<port> '` check.
func PortIsListening(ssOutput, port string) bool {
	needle := ":" + port
	for _, line := range strings.Split(ssOutput, "\n") {
		line = strings.TrimRight(line, " \t\r")
		if strings.HasPrefix(strings.TrimSpace(line), "State") {
			continue // header row
		}
		if strings.HasSuffix(line, needle) || strings.Contains(line, needle+" ") {
			return true
		}
	}
	return false
}

// LastLine returns the last non-empty line of a text file's contents.
func LastLine(content string) string {
	lines := strings.Split(content, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}

// ParseServiceStates pairs the unit names passed to `systemctl is-active` with
// the states it printed, one per line and in the same order. A unit systemctl
// said nothing about keeps an empty state.
func ParseServiceStates(names []string, output string) []models.ServiceState {
	var states []string
	for _, line := range strings.Split(output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			states = append(states, line)
		}
	}

	result := make([]models.ServiceState, 0, len(names))
	for i, name := range names {
		state := ""
		if i < len(states) {
			state = states[i]
		}
		result = append(result, models.ServiceState{Name: name, State: state})
	}
	return result
}

// FormatMemory condenses the "Mem:" row of `free -m` into one line. It returns
// an empty string when the row is missing or shaped differently than expected.
func FormatMemory(freeOutput string) string {
	for _, line := range strings.Split(freeOutput, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || !strings.HasPrefix(fields[0], "Mem") {
			continue
		}
		// total used free shared buff/cache available
		out := fmt.Sprintf("%s / %s MB used", fields[2], fields[1])
		if len(fields) >= 7 {
			out += fmt.Sprintf(", %s MB available", fields[6])
		}
		return out
	}
	return ""
}

// FormatDisk condenses the data row of `df -h /` into one line. It returns an
// empty string when there is no data row to read.
func FormatDisk(dfOutput string) string {
	for _, line := range strings.Split(dfOutput, "\n") {
		fields := strings.Fields(line)
		// Filesystem Size Used Avail Use% Mounted-on
		if len(fields) < 6 || fields[0] == "Filesystem" {
			continue
		}
		return fmt.Sprintf("%s / %s used (%s), %s free", fields[2], fields[1], fields[4], fields[3])
	}
	return ""
}

// FormatNodeStatus renders the node status screen. Values come from the host
// verbatim, so they are HTML-escaped before being embedded in the markup.
func FormatNodeStatus(status models.NodeStatus, now time.Time) string {
	var sb strings.Builder
	sb.WriteString("🖥 <b>Node Status</b>\n\n")

	channel := noData
	switch status.Channel {
	case models.ProbeUp:
		channel = "✅ listening"
	case models.ProbeDown:
		channel = "❌ not listening"
	case models.ProbeUnknown:
		channel = "❔ " + noData
	}
	sb.WriteString(fmt.Sprintf("🔌 <b>Channel %s:</b> %s\n", html.EscapeString(status.ChannelPort), channel))
	sb.WriteString(fmt.Sprintf("🎭 <b>REALITY donor:</b> %s\n", codeOrNoData(status.Dest)))
	sb.WriteString(fmt.Sprintf("🐕 <b>Watchdog:</b> %s\n", codeOrNoData(status.Watchdog)))

	sb.WriteString("\n⚙️ <b>Services</b>\n")
	if len(status.Services) == 0 {
		sb.WriteString(noData + "\n")
	}
	for _, service := range status.Services {
		icon := "❔"
		state := noData
		if service.State != "" {
			state = service.State
			icon = "❌"
			if service.IsActive() {
				icon = "✅"
			}
		}
		sb.WriteString(fmt.Sprintf("%s %s — %s\n", icon, html.EscapeString(service.Name), html.EscapeString(state)))
	}

	sb.WriteString("\n📦 <b>Host</b>\n")
	sb.WriteString(fmt.Sprintf("⏱ Uptime: %s\n", textOrNoData(status.Uptime)))
	sb.WriteString(fmt.Sprintf("🧠 Memory: %s\n", textOrNoData(status.Memory)))
	sb.WriteString(fmt.Sprintf("💾 Disk: %s\n", textOrNoData(status.Disk)))

	sb.WriteString(fmt.Sprintf("\n<i>Updated %s</i>", now.Format(constants.TimestampFormat)))
	return sb.String()
}

// FormatSubscriptionStats wraps the subscription statistics script output in a
// <code> block, dropping ANSI sequences and trimming it to Telegram's limit.
func FormatSubscriptionStats(output string) string {
	cleaned := strings.TrimSpace(StripANSI(output))
	if cleaned == "" {
		return "📡 <b>Subscriptions</b>\n\n⚠️ The statistics script returned nothing."
	}

	truncated := false
	if len(cleaned) > constants.MaxTelegramMessageLength {
		cleaned = cleaned[:constants.MaxTelegramMessageLength]
		truncated = true
	}

	var sb strings.Builder
	sb.WriteString("📡 <b>Subscriptions</b>\n\n<code>")
	sb.WriteString(html.EscapeString(cleaned))
	sb.WriteString("</code>")
	if truncated {
		sb.WriteString("\n\n<i>Output truncated.</i>")
	}
	return sb.String()
}

// textOrNoData escapes a value for HTML, or reports it as missing.
func textOrNoData(value string) string {
	if strings.TrimSpace(value) == "" {
		return noData
	}
	return html.EscapeString(value)
}

// codeOrNoData renders a value as inline code, or reports it as missing.
func codeOrNoData(value string) string {
	if strings.TrimSpace(value) == "" {
		return noData
	}
	return "<code>" + html.EscapeString(value) + "</code>"
}
