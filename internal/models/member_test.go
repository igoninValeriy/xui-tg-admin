package models

import (
	"testing"
	"time"
)

func TestSortMembersByCreationOrder(t *testing.T) {
	members := []MemberInfo{
		{BaseUsername: "c", ID: 3},
		{BaseUsername: "a", ID: 1},
		{BaseUsername: "b", ID: 2},
	}

	SortMembers(members, SortByCreationOrder)

	want := []int{1, 2, 3}
	for i, m := range members {
		if m.ID != want[i] {
			t.Errorf("position %d: got ID %d, want %d", i, m.ID, want[i])
		}
	}
}

func TestSortMembersByTrafficTotal(t *testing.T) {
	members := []MemberInfo{
		{BaseUsername: "low", TotalTraffic: 10},
		{BaseUsername: "high", TotalTraffic: 100},
		{BaseUsername: "mid", TotalTraffic: 50},
	}

	SortMembers(members, SortByTrafficTotal)

	want := []string{"high", "mid", "low"}
	for i, m := range members {
		if m.BaseUsername != want[i] {
			t.Errorf("position %d: got %q, want %q", i, m.BaseUsername, want[i])
		}
	}
}

func TestIsExpiredMember(t *testing.T) {
	now := time.Now().UnixMilli()

	cases := []struct {
		name       string
		expiryTime int64
		want       bool
	}{
		{"unlimited", 0, false},
		{"expired", now - 60_000, true},
		{"active", now + 60_000, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := MemberInfo{ExpiryTime: tc.expiryTime}
			if got := m.IsExpiredMember(); got != tc.want {
				t.Errorf("IsExpiredMember() with expiry %d = %v, want %v", tc.expiryTime, got, tc.want)
			}
		})
	}
}
