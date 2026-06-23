package helpers

import (
	"testing"

	"xui-tg-admin/internal/models"
)

const gb = int64(1) << 30

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1 << 10, "1.0 KB"},
		{1536, "1.5 KB"},
		{1 << 20, "1.0 MB"},
		{1 << 30, "1.00 GB"},
		{5 * gb, "5.00 GB"},
		{1 << 40, "1.00 TB"},
	}
	for _, c := range cases {
		if got := FormatBytes(c.in); got != c.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestAggregateTrafficGroupsBySubID(t *testing.T) {
	inbounds := []models.Inbound{
		{
			Remark:   "in1",
			Settings: `{"clients":[{"email":"alice-1","subId":"subA"},{"email":"bob-1","subId":"subB"}]}`,
			ClientStats: []models.ClientStat{
				{Email: "alice-1", Up: 1 * gb, Down: 2 * gb, Enable: true, ExpiryTime: 1000},
				{Email: "bob-1", Up: 0, Down: 10 * gb, Enable: false, ExpiryTime: 0},
			},
		},
		{
			Remark:   "in2",
			Settings: `{"clients":[{"email":"alice-2","subId":"subA"}]}`,
			ClientStats: []models.ClientStat{
				{Email: "alice-2", Up: 1 * gb, Down: 1 * gb, Enable: true, ExpiryTime: 2000},
			},
		},
	}

	report := AggregateTraffic(inbounds, []string{"alice-1"})

	if len(report.Users) != 2 {
		t.Fatalf("got %d users, want 2", len(report.Users))
	}

	// Sorted by total desc: bob (10 GB) before alice (5 GB).
	if report.Users[0].Name != "bob" || report.Users[1].Name != "alice" {
		t.Fatalf("unexpected order: %q, %q", report.Users[0].Name, report.Users[1].Name)
	}

	alice := report.Users[1]
	if alice.Up != 2*gb || alice.Down != 3*gb {
		t.Errorf("alice traffic = up %d down %d, want up %d down %d", alice.Up, alice.Down, 2*gb, 3*gb)
	}
	if !alice.Enabled {
		t.Error("alice should be enabled (one client enabled)")
	}
	if !alice.Online {
		t.Error("alice should be online")
	}
	if alice.ExpiryTime != 2000 {
		t.Errorf("alice expiry = %d, want 2000 (max)", alice.ExpiryTime)
	}

	bob := report.Users[0]
	if bob.Enabled {
		t.Error("bob should be disabled (no client enabled)")
	}
	if bob.Online {
		t.Error("bob should be offline")
	}

	if report.TotalUp != 2*gb || report.TotalDown != 13*gb {
		t.Errorf("totals = up %d down %d, want up %d down %d", report.TotalUp, report.TotalDown, 2*gb, 13*gb)
	}
	if report.OnlineCount != 1 {
		t.Errorf("online count = %d, want 1", report.OnlineCount)
	}

	if len(report.Inbounds) != 2 {
		t.Fatalf("got %d inbounds, want 2", len(report.Inbounds))
	}
	if report.Inbounds[0].Name != "in1" || report.Inbounds[0].Down != 12*gb || report.Inbounds[0].Up != 1*gb {
		t.Errorf("inbound[0] = %+v, want in1 down 12GB up 1GB", report.Inbounds[0])
	}
	if report.Inbounds[1].Name != "in2" {
		t.Errorf("inbound[1] = %q, want in2", report.Inbounds[1].Name)
	}
}

func TestAggregateTrafficFallsBackToBaseUsername(t *testing.T) {
	// No SubID in settings -> clients group by base username across inbounds.
	inbounds := []models.Inbound{
		{Remark: "in1", ClientStats: []models.ClientStat{{Email: "carl-1", Down: 1 * gb, Enable: true}}},
		{Remark: "in2", ClientStats: []models.ClientStat{{Email: "carl-2", Down: 2 * gb, Enable: true}}},
	}

	report := AggregateTraffic(inbounds, nil)

	if len(report.Users) != 1 {
		t.Fatalf("got %d users, want 1 (carl-1 and carl-2 merge)", len(report.Users))
	}
	carl := report.Users[0]
	if carl.Name != "carl" {
		t.Errorf("name = %q, want carl", carl.Name)
	}
	if carl.Down != 3*gb {
		t.Errorf("carl down = %d, want %d", carl.Down, 3*gb)
	}
}
