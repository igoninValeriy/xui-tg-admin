package render

import (
	"bytes"
	"image/png"
	"testing"
	"time"

	"xui-tg-admin/internal/helpers"
)

func TestTrafficReportProducesValidPNG(t *testing.T) {
	gb := int64(1) << 30
	report := helpers.TrafficReport{
		Users: []helpers.UserTraffic{
			{Name: "alice", Down: 5 * gb, Up: gb, Enabled: true, Online: true},
			{Name: "bob", Down: gb, Enabled: false},
		},
		Inbounds: []helpers.InboundTraffic{
			{Name: "in1", Down: 4 * gb, Up: gb},
			{Name: "in2", Down: 2 * gb},
		},
		TotalDown:   6 * gb,
		TotalUp:     gb,
		OnlineCount: 1,
	}

	img, err := TrafficReport(report, time.Unix(1750000000, 0))
	if err != nil {
		t.Fatalf("TrafficReport returned error: %v", err)
	}
	if len(img) == 0 {
		t.Fatal("TrafficReport returned empty image")
	}
	if _, err := png.Decode(bytes.NewReader(img)); err != nil {
		t.Fatalf("output is not a valid PNG: %v", err)
	}
}

// An empty report (no users, no inbounds) must still render without panicking.
func TestTrafficReportEmpty(t *testing.T) {
	img, err := TrafficReport(helpers.TrafficReport{}, time.Unix(1750000000, 0))
	if err != nil {
		t.Fatalf("TrafficReport returned error: %v", err)
	}
	if _, err := png.Decode(bytes.NewReader(img)); err != nil {
		t.Fatalf("output is not a valid PNG: %v", err)
	}
}
