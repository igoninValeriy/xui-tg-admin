package helpers

import (
	"strings"
	"testing"
	"time"

	"xui-tg-admin/internal/models"
)

const ssSample = `State  Recv-Q Send-Q Local Address:Port  Peer Address:Port
LISTEN 0      8192       127.0.0.1:2096       0.0.0.0:*
LISTEN 0      4096         0.0.0.0:22         0.0.0.0:*
LISTEN 0      8192               *:443              *:*
`

func TestPortIsListening(t *testing.T) {
	if !PortIsListening(ssSample, "443") {
		t.Error("expected port 443 to be reported as listening")
	}
	if PortIsListening(ssSample, "8443") {
		t.Error("expected port 8443 to be reported as not listening")
	}
	if PortIsListening(ssSample, "80") {
		t.Error("expected port 80 not to match the :8080-style substrings")
	}
	if PortIsListening("", "443") {
		t.Error("expected empty output to report nothing listening")
	}
}

func TestPortIsListeningIgnoresHeader(t *testing.T) {
	if PortIsListening("State Recv-Q Send-Q Local Address:Port\n", "443") {
		t.Error("header row must not be treated as a socket")
	}
}

func TestLastLine(t *testing.T) {
	if got := LastLine("a\nb\nc\n\n"); got != "c" {
		t.Errorf("expected last non-empty line 'c', got %q", got)
	}
	if got := LastLine("   \n"); got != "" {
		t.Errorf("expected empty result for blank content, got %q", got)
	}
}

func TestParseServiceStates(t *testing.T) {
	names := []string{"x-ui", "vpn-watchdog.timer", "fail2ban"}
	states := ParseServiceStates(names, "active\ninactive\n")

	if len(states) != 3 {
		t.Fatalf("expected 3 states, got %d", len(states))
	}
	if !states[0].IsActive() {
		t.Error("x-ui should be active")
	}
	if states[1].State != "inactive" {
		t.Errorf("expected inactive, got %q", states[1].State)
	}
	if states[2].State != "" {
		t.Errorf("missing line must leave the state empty, got %q", states[2].State)
	}
}

func TestFormatMemory(t *testing.T) {
	out := `               total        used        free      shared  buff/cache   available
Mem:            3848         558        2101           5        1461        3289
Swap:           2047           0        2047`

	got := FormatMemory(out)
	if !strings.Contains(got, "558 / 3848 MB used") || !strings.Contains(got, "3289 MB available") {
		t.Errorf("unexpected memory line: %q", got)
	}
	if FormatMemory("garbage") != "" {
		t.Error("unparsable output must yield an empty string")
	}
}

func TestFormatDisk(t *testing.T) {
	out := `Filesystem      Size  Used Avail Use% Mounted on
/dev/sda1        58G  4.8G   53G   9% /`

	got := FormatDisk(out)
	if !strings.Contains(got, "4.8G / 58G used (9%)") || !strings.Contains(got, "53G free") {
		t.Errorf("unexpected disk line: %q", got)
	}
	if FormatDisk("Filesystem      Size  Used Avail Use% Mounted on\n") != "" {
		t.Error("a header-only listing must yield an empty string")
	}
}

func TestStripANSI(t *testing.T) {
	if got := StripANSI("\x1b[31mred\x1b[0m plain"); got != "red plain" {
		t.Errorf("expected escapes to be removed, got %q", got)
	}
}

func TestFormatNodeStatusReportsMissingData(t *testing.T) {
	status := models.NodeStatus{
		ChannelPort: "443",
		Channel:     models.ProbeUnknown,
		Services:    []models.ServiceState{{Name: "x-ui"}},
	}

	text := FormatNodeStatus(status, time.Unix(0, 0).UTC())
	for _, field := range []string{"REALITY donor", "Watchdog", "Uptime", "Memory", "Disk"} {
		if !strings.Contains(text, field) {
			t.Errorf("expected the screen to mention %q", field)
		}
	}
	if strings.Count(text, noData) < 6 {
		t.Errorf("every failed probe must be shown as %q, got:\n%s", noData, text)
	}
}

func TestFormatNodeStatusEscapesHostValues(t *testing.T) {
	status := models.NodeStatus{
		ChannelPort: "443",
		Channel:     models.ProbeUp,
		Dest:        "<b>evil</b>",
		Watchdog:    "check dest=a & b",
		Uptime:      "up 7 hours",
	}

	text := FormatNodeStatus(status, time.Unix(0, 0).UTC())
	if strings.Contains(text, "<b>evil</b>") {
		t.Error("values read from the host must be HTML-escaped")
	}
	if !strings.Contains(text, "&amp;") {
		t.Error("ampersands from the host must be escaped")
	}
	if !strings.Contains(text, "listening") {
		t.Error("an up channel must be reported as listening")
	}
}

func TestFormatSubscriptionStats(t *testing.T) {
	got := FormatSubscriptionStats("\x1b[32mphone   requests: 5\x1b[0m")
	if strings.Contains(got, "\x1b") {
		t.Error("ANSI sequences must be stripped")
	}
	if !strings.Contains(got, "<code>") || !strings.Contains(got, "</code>") {
		t.Errorf("output must be wrapped in a code block: %q", got)
	}

	empty := FormatSubscriptionStats("   \n")
	if !strings.Contains(empty, "returned nothing") {
		t.Errorf("empty output must be reported, got %q", empty)
	}

	long := FormatSubscriptionStats(strings.Repeat("x", 10000))
	if !strings.Contains(long, "truncated") {
		t.Error("oversized output must be truncated and say so")
	}
}
