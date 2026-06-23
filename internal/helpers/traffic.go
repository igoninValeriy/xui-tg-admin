package helpers

import (
	"fmt"
	"sort"

	"xui-tg-admin/internal/models"
)

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
